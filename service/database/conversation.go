package database

import (
	"database/sql"
	"errors"
	"strings"
)

func (db *appdbimpl) GetMyConversations(userID string) ([]Conversation, error) {
	var rows *sql.Rows
	var err error

	// SQL Logic:
	// 1. Go to 'participants' table.
	// 2. Find rows where user_id matches me.
	// 3. Join with 'conversations' table to get the chat details.
	// 4. Order by 'updated_at'	// Returns all conversations where the user is a participant
	rows, err = db.c.Query(`
		SELECT c.conversation_id, c.name, c.photo_id, c.last_message, c.updated_at,
		(SELECT COUNT(*) 
		 FROM messages m 
		 WHERE m.conversation_id = c.conversation_id 
		 AND m.created_at > p.last_seen
		) as unread_count
		FROM conversations c
		JOIN participants p ON c.conversation_id = p.conversation_id
		WHERE p.user_id = ?
		ORDER BY c.updated_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []Conversation
	for rows.Next() {
		var c Conversation
		var name, photoID, lastMessage, updatedAt sql.NullString
		var unreadCount int

		if err := rows.Scan(&c.ID, &name, &photoID, &lastMessage, &updatedAt, &unreadCount); err != nil {
			return nil, err
		}

		// ...

		if name.Valid && strings.TrimSpace(name.String) != "" {
			c.Name = name.String
			c.IsGroup = true
			if photoID.Valid && photoID.String != "" {
				c.Photo = "/images/" + photoID.String
			}
		} else {
			c.IsGroup = false
		}
		// ...
		if lastMessage.Valid {
			c.LastMessage = lastMessage.String
		}
		if updatedAt.Valid {
			c.UpdatedAt = updatedAt.String
		}
		c.UnreadCount = unreadCount

		// If name is empty, it's a Direct Message. We need to find the other person's name.
		if !c.IsGroup {
			var otherUsername string
			var otherPhotoID sql.NullString
			// Find a participant in this conversation who is NOT me.
			err := db.c.QueryRow(`
				SELECT u.username, u.photo_id
				FROM participants p
				JOIN users u ON p.user_id = u.identifier
				WHERE p.conversation_id = ? AND p.user_id != ?
				LIMIT 1
			`, c.ID, userID).Scan(&otherUsername, &otherPhotoID)

			if err == nil {
				c.Name = otherUsername
				if otherPhotoID.Valid && otherPhotoID.String != "" {
					c.Photo = "/images/" + otherPhotoID.String
				}
			} else {
				if errors.Is(err, sql.ErrNoRows) {
					// Fallback: This might be a "Self Chat" (me talking to me)
					// So let's check if *I* am in there (which I should be, otherwise I wouldn't see it).
					// We query for MY username/photo.
					errMy := db.c.QueryRow(`
						SELECT u.username, u.photo_id
						FROM participants p
						JOIN users u ON p.user_id = u.identifier
						WHERE p.conversation_id = ? AND p.user_id = ?
						LIMIT 1
					`, c.ID, userID).Scan(&otherUsername, &otherPhotoID)

					if errMy == nil {
						c.Name = otherUsername + " (You)"
						if otherPhotoID.Valid && otherPhotoID.String != "" {
							c.Photo = "/images/" + otherPhotoID.String
						}
					} else {
						c.Name = "Unknown Chat"
					}
				} else {
					c.Name = "ERR: " + err.Error()
				}
			}
		}

		chats = append(chats, c)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return chats, nil
}

func (db *appdbimpl) CreateOneToOneConversation(myID string, targetUsername string) (Conversation, bool, error) {
	var targetID string

	// 1. Find the Target User's ID
	err := db.c.QueryRow("SELECT identifier FROM users WHERE username = ?", targetUsername).Scan(&targetID)
	if errors.Is(err, sql.ErrNoRows) {
		// We return a specific error so the API knows to send 404
		return Conversation{}, false, errors.New("target user not found")
	}
	if err != nil {
		return Conversation{}, false, err
	}

	// 2. Check if a 1-on-1 conversation already exists
	// A 1-on-1 chat is defined as a chat where BOTH are participants AND the 'name' is NULL (group chats usually have names)
	var existingID string
	err = db.c.QueryRow(`
		SELECT c.conversation_id 
		FROM conversations c
		JOIN participants p1 ON c.conversation_id = p1.conversation_id
		JOIN participants p2 ON c.conversation_id = p2.conversation_id
		WHERE p1.user_id = ? AND p2.user_id = ? AND c.name IS NULL
	`, myID, targetID).Scan(&existingID)

	// CASE A: Conversation Exists -> Return it (Status 200)
	if err == nil {
		return Conversation{ID: existingID}, false, nil
	}

	// CASE B: Create New Conversation (Status 201)
	newID, err := GenerateNewID() // Use the helper from database.go
	if err != nil {
		return Conversation{}, false, err
	}

	// Start a simplified flow (in real apps, use transactions!)

	// Insert Conversation
	_, err = db.c.Exec("INSERT INTO conversations (conversation_id, name) VALUES (?, NULL)", newID)
	if err != nil {
		return Conversation{}, false, err
	}

	// Insert Participant 1 (Me)
	_, err = db.c.Exec("INSERT INTO participants (conversation_id, user_id) VALUES (?, ?)", newID, myID)
	if err != nil {
		return Conversation{}, false, err
	}

	// Insert Participant 2 (Target)
	_, err = db.c.Exec("INSERT INTO participants (conversation_id, user_id) VALUES (?, ?)", newID, targetID)
	if err != nil {
		return Conversation{}, false, err
	}

	// Return the new object
	return Conversation{
		ID:          newID,
		Name:        "", // Empty for 1-on-1
		LastMessage: "",
		UpdatedAt:   "", // DB defaults this, but we can return empty or fetch it
	}, true, nil
}

func (db *appdbimpl) ClearConversationHistory(conversationID string, myID string) error {

	// 1. Check if the conversation exists and I am a participant
	// We try to update the row.
	res, err := db.c.Exec(`
        UPDATE participants 
        SET last_seen = CURRENT_TIMESTAMP 
        WHERE conversation_id = ? AND user_id = ?`,
		conversationID, myID)

	if err != nil {
		return err
	}

	// 2. Check if we actually updated anything
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	// If 0 rows affected, it means the conversation ID is wrong
	// OR I am not a participant.
	if rowsAffected == 0 {
		return ErrConversationNotFound
	}

	return nil
}

func (db *appdbimpl) CreateGroup(name string, creatorID string, memberUsernames []string) (Conversation, error) {

	// 1. Generate new Group ID
	groupID, err := GenerateNewID() // Use the public helper with Capital G
	if err != nil {
		return Conversation{}, err
	}

	// 2. Create the Conversation (With a Name)
	// Unlike 1-on-1, "name" is NOT NULL here.
	_, err = db.c.Exec("INSERT INTO conversations (conversation_id, name) VALUES (?, ?)", groupID, name)
	if err != nil {
		return Conversation{}, err
	}

	// 3. Add the Creator (Me) to participants
	_, err = db.c.Exec("INSERT INTO participants (conversation_id, user_id) VALUES (?, ?)", groupID, creatorID)
	if err != nil {
		return Conversation{}, err
	}

	// 4. Add the other Members
	for _, username := range memberUsernames {
		// A. Find User ID
		var memberID string
		err := db.c.QueryRow("SELECT identifier FROM users WHERE username = ?", username).Scan(&memberID)

		// If user doesn't exist, we skip them (or you could return an error)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return Conversation{}, err
		}

		// B. Add to Participants
		// We use INSERT OR IGNORE just in case the same user is listed twice
		_, err = db.c.Exec("INSERT OR IGNORE INTO participants (conversation_id, user_id) VALUES (?, ?)", groupID, memberID)
		if err != nil {
			return Conversation{}, err
		}
	}

	// 5. Return the result
	return Conversation{
		ID:          groupID,
		Name:        name,
		LastMessage: "", // Empty for new group
		UpdatedAt:   "", // DB defaults this
	}, nil
}

func (db *appdbimpl) GetGroupDetails(groupID string, myID string) (GroupDetails, error) {

	var g GroupDetails
	g.ID = groupID

	// 1. Get Group Info & Verify Membership
	// We join participants to check if 'myID' is in the group.
	// If 'name' is NULL (meaning it's a 1-on-1 chat, not a group), we might want to handle that.
	// The query assumes we only want "Named Groups".
	var photoID sql.NullString
	err := db.c.QueryRow(`
		SELECT c.name, c.photo_id 
		FROM conversations c
		JOIN participants p ON c.conversation_id = p.conversation_id
		WHERE c.conversation_id = ? AND p.user_id = ? AND c.name IS NOT NULL`,
		groupID, myID).Scan(&g.Name, &photoID)

	if errors.Is(err, sql.ErrNoRows) {
		return GroupDetails{}, errors.New("group not found or access denied")
	}
	if err != nil {
		return GroupDetails{}, err
	}
	g.PhotoID = photoID.String

	// 2. Get All Members
	rows, err := db.c.Query(`
        SELECT u.username 
        FROM users u
        JOIN participants p ON u.identifier = p.user_id
        WHERE p.conversation_id = ?`, groupID)
	if err != nil {
		return GroupDetails{}, err
	}
	defer rows.Close()

	g.Members = []string{}
	for rows.Next() {
		var username string
		if err := rows.Scan(&username); err != nil {
			return GroupDetails{}, err
		}
		g.Members = append(g.Members, username)
	}
	if rows.Err() != nil {
		return GroupDetails{}, rows.Err()
	}

	return g, nil
}

func (db *appdbimpl) LeaveGroup(groupID string, userID string) error {

	// Execute Delete
	res, err := db.c.Exec("DELETE FROM participants WHERE conversation_id = ? AND user_id = ?", groupID, userID)
	if err != nil {
		return err
	}

	// Check if we actually removed anyone
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		// This implies either the group doesn't exist
		// OR the user wasn't a member of it.
		return errors.New("group not found or user not a member")
	}

	return nil
}

func (db *appdbimpl) AddUserToGroup(groupID string, myID string, targetUsername string) error {

	// 1. Permission Check: Am I (myID) a member of this group?
	// And does the group exist?
	var dummy int
	err := db.c.QueryRow(`
		SELECT 1 FROM participants 
		WHERE conversation_id = ? AND user_id = ?`,
		groupID, myID).Scan(&dummy)

	if errors.Is(err, sql.ErrNoRows) {
		// If I'm not in it, either group doesn't exist or I'm forbidden.
		// We return a specific error to handle 403 vs 404 in the handler,
		// or just Forbidden if we want to be safe.
		return errors.New("forbidden: access denied or group not found")
	}
	if err != nil {
		return err
	}

	// 2. Find Target User ID
	var targetID string
	err = db.c.QueryRow("SELECT identifier FROM users WHERE username = ?", targetUsername).Scan(&targetID)
	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("target user not found")
	}
	if err != nil {
		return err
	}

	// 3. Check Conflict: Is target user already in the group?
	err = db.c.QueryRow(`
        SELECT 1 FROM participants 
        WHERE conversation_id = ? AND user_id = ?`,
		groupID, targetID).Scan(&dummy)

	if err == nil {
		// Row exists -> Conflict
		return errors.New("conflict: user already in group")
	}
	if !errors.Is(err, sql.ErrNoRows) {
		// Database error
		return err
	}

	// 4. Add the User
	_, err = db.c.Exec("INSERT INTO participants (conversation_id, user_id) VALUES (?, ?)", groupID, targetID)
	return err
}

// Am I in this group? Is this name taken by another group? Change the name.
func (db *appdbimpl) SetGroupName(groupID string, myID string, newName string) error {

	// 1. Permission Check: Am I a member?
	var dummy int
	err := db.c.QueryRow(`
		SELECT 1 FROM participants 
		WHERE conversation_id = ? AND user_id = ?`,
		groupID, myID).Scan(&dummy)

	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("forbidden: access denied or group not found")
	}
	if err != nil {
		return err
	}

	// 2. Uniqueness Check: Is this name already taken by *another* group?
	// We check if a conversation exists with 'name = newName' AND 'conversation_id != groupID'
	err = db.c.QueryRow(`
        SELECT 1 FROM conversations 
        WHERE name = ? AND conversation_id != ?`,
		newName, groupID).Scan(&dummy)

	if err == nil {
		return errors.New("conflict: group name already taken")
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	// 3. Update the Name
	_, err = db.c.Exec("UPDATE conversations SET name = ? WHERE conversation_id = ?", newName, groupID)
	return err
}

// verifies membership and then updates the photo_id column.
func (db *appdbimpl) SetGroupPhoto(groupID string, myID string, photoID string) error {

	// 1. Permission Check: Am I a member of this group?
	// And does the group exist? (Group implies name is not null usually, but basic participants check is enough)
	var dummy int
	err := db.c.QueryRow(`
		SELECT 1 FROM participants 
		WHERE conversation_id = ? AND user_id = ?`,
		groupID, myID).Scan(&dummy)

	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("forbidden: access denied or group not found")
	}
	if err != nil {
		return err
	}

	// 2. Update the Photo ID
	_, err = db.c.Exec("UPDATE conversations SET photo_id = ? WHERE conversation_id = ?", photoID, groupID)
	return err
}
