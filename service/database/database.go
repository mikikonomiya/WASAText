/*
Package database is the middleware between the app database and the code. All data (de)serialization (save/load) from a
persistent database are handled here. Database specific logic should never escape this package.

To use this package you need to apply migrations to the database if needed/wanted, connect to it (using the database
data source name from config), and then initialize an instance of AppDatabase from the DB connection.

For example, this code adds a parameter in `webapi` executable for the database data source name (add it to the
main.WebAPIConfiguration structure):

	DB struct {
		Filename string `conf:""`
	}

This is an example on how to migrate the DB and connect to it:

	// Start Database
	logger.Println("initializing database support")
	db, err := sql.Open("sqlite3", "./foo.db")
	if err != nil {
		logger.WithError(err).Error("error opening SQLite DB")
		return fmt.Errorf("opening SQLite: %w", err)
	}
	defer func() {
		logger.Debug("database stopping")
		_ = db.Close()
	}()

Then you can initialize the AppDatabase and pass it to the api package.
*/
package database

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
)

// AppDatabase is the high level interface for the DB
type AppDatabase interface {

	// create new user if username doesnt exist. returns (identifier, error)
	GetOrCreateUser(username string) (string, error)
	// returns a list of users matching the query (if empty, returns all users)
	SearchUsers(query string) ([]User, error)
	// returns the user structure
	GetUser(username string) (User, error)
	// update username for given user ID
	SetUsername(id string, newName string) error
	// update user's record to show they have a photo
	SetUserPhoto(id string, photoID string) error
	// returns all chats where user is a participant, sorted (newest first)
	GetMyConversations(userID string) ([]Conversation, error)
	// finds an existng 1-to-1 chat or creates a new one
	// returns the conversation, bool (true if new chat created, false if existing found)
	// and err (user not found)
	CreateOneToOneConversation(myID string, targetUsername string) (Conversation, bool, error)
	// returns messages if the user is a participant
	GetConversationMessages(conversationID string, myID string) ([]Message, error)
	// updates user's "last_seen" timestamp, hiding all previous messages
	ClearConversationHistory(conversationID string, myID string) error
	// adds a new text message to conversation and updates conversation's "last_message" and "updated_at" fields.
	SendMessage(conversationID string, senderID string, text string) (Message, error)
	// adds a new image message. content is the filename (e.g., "msg-1234.png").
	SendPhotoMessage(conversationID string, senderID string, photoID string) error
	// returns a list of photo filenames for the chat e.g., ["img1.png", "img2.png"]
	GetConversationPhotos(conversationID string, myID string) ([]string, error)
	// returns messages from a conversation that the user has starred.
	GetStarredMessages(conversationID string, myID string) ([]Message, error)
	// SearchMessages searches for text in all conversations where the user is a participant.
	SearchMessages(myID string, query string) ([]Message, error)
	// copies content from one message to a new message in a destination chat.
	// and checks permissions for both source (read) and destination (write).
	ForwardMessage(myID string, sourceMsgID string, destConvID string) error
	// deletes a message. scope can be "me" (hide for user) or "everyone" (hard delete).
	DeleteMessage(messageID string, myID string, scope string) error
	// GetAllStarredMessages returns ALL messages starred by the user across all chats.
	GetAllStarredMessages(myID string) ([]Message, error)
	// StarMessage marks a message as starred for the user.
	StarMessage(messageID string, myID string) error
	// Returns error if the message wasn't starred or doesn't exist.
	UnstarMessage(messageID string, myID string) error
	// ReactToMessage adds or updates a user's reaction to a message.
	ReactToMessage(messageID string, myID string, emoji string) error
	// UnreactToMessage removes the user's reaction from a message.
	UnreactToMessage(messageID string, myID string) error
	// CreateGroup creates a named conversation and adds the creator + a list of users.
	CreateGroup(name string, creatorID string, memberUsernames []string) (Conversation, error)
	// GetGroupDetails returns info + member list for a group.
	GetGroupDetails(groupID string, myID string) (GroupDetails, error)
	// removes the user from the specified group. returns error if group not found or user wasn't a member.
	LeaveGroup(groupID string, userID string) error
	// adds a target user to a group, checks if 'myID' is allowed to add people.
	AddUserToGroup(groupID string, myID string, targetUsername string) error
	// updates the name of a group. checks for permissions (membership) and uniqueness (name collision).
	SetGroupName(groupID string, myID string, newName string) error
	// updates the profile picture of a group. checks if 'myID' is a member before updating.
	SetGroupPhoto(groupID string, myID string, photoID string) error
	Ping() error
}

type appdbimpl struct {
	c *sql.DB
}

// New returns a new instance of AppDatabase based on the SQLite connection `db`.
// `db` is required - an error will be returned if `db` is `nil`.
func New(db *sql.DB) (AppDatabase, error) {
	if db == nil {
		return nil, errors.New("database is required when building a AppDatabase")
	}

	// Check if table exists. If not, the database is empty, and we need to create the structure
	var tableName string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='users';`).Scan(&tableName)

	if errors.Is(err, sql.ErrNoRows) {
		// Users Table
		sqlStmt := `CREATE TABLE IF NOT EXISTS users (
		identifier TEXT NOT NULL PRIMARY KEY, 
		username TEXT NOT NULL UNIQUE,
		photo_id TEXT DEFAULT NULL 
		);`
		if err != nil {
			return nil, fmt.Errorf("error creating users table: %w", err)
		}

		// Conversations Table
		sqlStmt = `CREATE TABLE IF NOT EXISTS conversations (
			conversation_id TEXT NOT NULL PRIMARY KEY,
			name TEXT, 
			photo_id TEXT, 
			last_message TEXT, 
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, err
		}

		// Participants Table
		sqlStmt = `CREATE TABLE IF NOT EXISTS participants (
		conversation_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		last_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (conversation_id, user_id),
		FOREIGN KEY (conversation_id) REFERENCES conversations(conversation_id),
		FOREIGN KEY (user_id) REFERENCES users(identifier)
		);`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, err
		}

		// Messages Table
		sqlStmt = `CREATE TABLE IF NOT EXISTS messages (
			message_id TEXT NOT NULL PRIMARY KEY,
			conversation_id TEXT NOT NULL,
			sender_id TEXT NOT NULL,
			content TEXT NOT NULL, -- The text message or the filename of the image
			kind TEXT NOT NULL,    -- 'text' or 'image'
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (conversation_id) REFERENCES conversations(conversation_id),
			FOREIGN KEY (sender_id) REFERENCES users(identifier)
		);`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, err
		}

		// Stars Table (NEW)
		sqlStmt = `CREATE TABLE IF NOT EXISTS stars (
			user_id TEXT NOT NULL,
			message_id TEXT NOT NULL,
			PRIMARY KEY (user_id, message_id),
			FOREIGN KEY (user_id) REFERENCES users(identifier),
			FOREIGN KEY (message_id) REFERENCES messages(message_id)
		);`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating stars table: %w", err)
		}

		//  Deleted Messages Table (NEW)
		// Keeps track of messages hidden by specific users (scope="me")
		sqlStmt = `CREATE TABLE IF NOT EXISTS deleted_messages (
			user_id TEXT NOT NULL,
			message_id TEXT NOT NULL,
			PRIMARY KEY (user_id, message_id),
			FOREIGN KEY (user_id) REFERENCES users(identifier),
			FOREIGN KEY (message_id) REFERENCES messages(message_id)
		);`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating deleted_messages table: %w", err)
		}

		// Reactions Table (NEW)
		sqlStmt = `CREATE TABLE IF NOT EXISTS reactions (
			user_id TEXT NOT NULL,
			message_id TEXT NOT NULL,
			emoji TEXT NOT NULL,
			PRIMARY KEY (user_id, message_id), -- Ensure one reaction per user per message
			FOREIGN KEY (user_id) REFERENCES users(identifier),
			FOREIGN KEY (message_id) REFERENCES messages(message_id)
		);`
		_, err = db.Exec(sqlStmt)
		if err != nil {
			return nil, fmt.Errorf("error creating reactions table: %w", err)
		}
	}

	return &appdbimpl{
		c: db,
	}, nil
}

func (db *appdbimpl) Ping() error {
	return db.c.Ping()
}

// helper function to generate random string of 16 char
func GenerateNewID() (string, error) {
	bytes := make([]byte, 8) // 8 bytes = 16 hex characters
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// User struct represents an user object in the database
type User struct {
	Identifier string `json:"identifier"`
	Username   string `json:"username"`
}

type Conversation struct {
	ID          string `json:"id"`
	Name        string `json:"name,omitempty"` // "omitempty" hides this field if empty
	LastMessage string `json:"last_message"`
	UpdatedAt   string `json:"updated_at"`
}

type Message struct {
	ID             string `json:"id"`
	ConversationID string `json:"conversationId"`
	SenderID       string `json:"senderId"`
	Content        string `json:"content"`
	Kind           string `json:"kind"` // "text" or "image"
	CreatedAt      string `json:"created_at"`
}

type GroupDetails struct {
	ID      string   `json:"groupId"`
	Name    string   `json:"groupName"`
	PhotoID string   `json:"groupPhoto"` // We store the ID/filename here, API adds the URL prefix
	Members []string `json:"members"`
}
