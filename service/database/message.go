package database

import (
	"database/sql"
	"errors"
)

// Matches GetConversationMessages
func (db *appdbimpl) GetConversationMessages(conversationID string, myID string) ([]Message, error) {

	// 1. Security Check
	var dummy int
	err := db.c.QueryRow(`SELECT 1 FROM participants WHERE conversation_id = ? AND user_id = ?`, conversationID, myID).Scan(&dummy)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrConversationNotFound
	}
	if err != nil {
		return nil, err
	}

	// 2. Fetch Messages
	// We utilize LEFT JOIN to find if the message is in deleted_messages.
	// Then 'WHERE d.message_id IS NULL' filters those out.
	rows, err := db.c.Query(`
		SELECT m.message_id, m.conversation_id, m.sender_id, m.content, m.kind, m.created_at 
		FROM messages m
		JOIN participants p ON m.conversation_id = p.conversation_id
        LEFT JOIN deleted_messages d ON m.message_id = d.message_id AND d.user_id = ?
		WHERE m.conversation_id = ? 
          AND p.user_id = ? 
          AND m.created_at > p.last_seen
          AND d.message_id IS NULL  
		ORDER BY m.created_at DESC`,
		myID, conversationID, myID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Content, &m.Kind, &m.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return messages, nil
}

// check if you are a participant, generate a new Message ID,
// insert the message and update the parent conversation (s.t."Last Message" preview changes).
func (db *appdbimpl) SendMessage(conversationID string, senderID string, text string) (Message, error) {

	// 1. Security Check: Are you a participant?
	var dummy int
	err := db.c.QueryRow("SELECT 1 FROM participants WHERE conversation_id = ? AND user_id = ?", conversationID, senderID).Scan(&dummy)
	if errors.Is(err, sql.ErrNoRows) {
		return Message{}, ErrConversationNotFound
	}
	if err != nil {
		return Message{}, err
	}

	// 2. Generate ID
	messageID, err := GenerateNewID()
	if err != nil {
		return Message{}, err
	}

	// 3. Insert Message
	// kind is 'text'
	_, err = db.c.Exec(`
		INSERT INTO messages (message_id, conversation_id, sender_id, content, kind, created_at) 
		VALUES (?, ?, ?, ?, 'text', CURRENT_TIMESTAMP)`,
		messageID, conversationID, senderID, text)
	if err != nil {
		return Message{}, err
	}

	// 4. Update Conversation Metadata (Last Message & Time)
	// We truncate text to 50 chars for the preview to save space (optional, but good practice)
	preview := text
	if len(preview) > 50 {
		preview = preview[:47] + "..."
	}
	_, err = db.c.Exec(`
		UPDATE conversations 
		SET last_message = ?, updated_at = CURRENT_TIMESTAMP 
		WHERE conversation_id = ?`,
		preview, conversationID)
	if err != nil {
		return Message{}, err
	}

	// 5. Return the full object
	// We need to fetch the 'created_at' time that the DB just generated
	var createdAt string
	err = db.c.QueryRow("SELECT created_at FROM messages WHERE message_id = ?", messageID).Scan(&createdAt)
	if err != nil {
		return Message{}, err
	}

	return Message{
		ID:             messageID,
		ConversationID: conversationID,
		SenderID:       senderID,
		Content:        text,
		Kind:           "text",
		CreatedAt:      createdAt,
	}, nil
}

func (db *appdbimpl) SendPhotoMessage(conversationID string, senderID string, photoID string) error {

	// 1. Security Check: Are you a participant?
	var dummy int
	err := db.c.QueryRow("SELECT 1 FROM participants WHERE conversation_id = ? AND user_id = ?", conversationID, senderID).Scan(&dummy)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrConversationNotFound
	}
	if err != nil {
		return err
	}

	// 2. Generate Message ID
	// Note: We often use the photoID (filename) as the messageID or generate a new one.
	// Here we generate a new ID for the message itself.
	messageID, err := GenerateNewID()
	if err != nil {
		return err
	}

	// 3. Insert Message
	// kind is 'image', content is the filename (photoID)
	_, err = db.c.Exec(`
		INSERT INTO messages (message_id, conversation_id, sender_id, content, kind, created_at) 
		VALUES (?, ?, ?, ?, 'image', CURRENT_TIMESTAMP)`,
		messageID, conversationID, senderID, photoID)
	if err != nil {
		return err
	}

	// 4. Update Conversation Metadata
	// For images, the "last message" preview is usually "[Image]" or "Photo"
	_, err = db.c.Exec(`
		UPDATE conversations 
		SET last_message = '[Photo]', updated_at = CURRENT_TIMESTAMP 
		WHERE conversation_id = ?`,
		conversationID)

	return err
}

// This query is very similar to GetConversationMessages but simpler: we only want the
// content column (the filename) and we filter by kind = 'image'
func (db *appdbimpl) GetConversationPhotos(conversationID string, myID string) ([]string, error) {

	// 1. Security Check: Is the user a participant?
	var dummy int
	err := db.c.QueryRow("SELECT 1 FROM participants WHERE conversation_id = ? AND user_id = ?", conversationID, myID).Scan(&dummy)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrConversationNotFound
	}
	if err != nil {
		return nil, err
	}

	// 2. Fetch Filenames
	// We only select 'content' where kind='image'
	rows, err := db.c.Query(`
		SELECT content 
		FROM messages 
		WHERE conversation_id = ? AND kind = 'image'
		ORDER BY created_at DESC`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 3. Parse Results
	var photos []string
	for rows.Next() {
		var filename string
		if err := rows.Scan(&filename); err != nil {
			return nil, err
		}
		photos = append(photos, filename)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return photos, nil
}

// Check if I am a participant in the conversation, Join messages with stars to find only the ones I marked
// and Filter by conversation_id (so we don't return starred messages from other chats)
func (db *appdbimpl) GetStarredMessages(conversationID string, myID string) ([]Message, error) {

	// 1. Security Check: Participant?
	var dummy int
	err := db.c.QueryRow("SELECT 1 FROM participants WHERE conversation_id = ? AND user_id = ?", conversationID, myID).Scan(&dummy)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrConversationNotFound
	}
	if err != nil {
		return nil, err
	}

	// 2. Fetch Starred Messages
	// Logic: Get messages M where there is a row in STARS table for (myID, m.message_id)
	// AND the message belongs to this conversationID.
	query := `
		SELECT m.message_id, m.conversation_id, m.sender_id, m.content, m.kind, m.created_at
		FROM messages m
		JOIN stars s ON m.message_id = s.message_id
		WHERE s.user_id = ? AND m.conversation_id = ?
		ORDER BY m.created_at DESC`

	rows, err := db.c.Query(query, myID, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 3. Parse Results
	var messages []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Content, &m.Kind, &m.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return messages, nil
}
func (db *appdbimpl) SearchMessages(myID string, query string) ([]Message, error) {

	// We want messages where:
	// 1. The user (myID) is a participant in the conversation.
	// 2. The message content contains the query string.
	// 3. The kind is 'text' (usually we don't search image filenames, but you could remove this check if you want).
	sqlQuery := `
		SELECT m.message_id, m.conversation_id, m.sender_id, m.content, m.kind, m.created_at
		FROM messages m
		JOIN participants p ON m.conversation_id = p.conversation_id
		WHERE p.user_id = ? 
		  AND m.kind = 'text'
		  AND m.content LIKE ?
		ORDER BY m.created_at DESC
	`

	// Add wildcards for partial matching
	searchTerm := "%" + query + "%"

	rows, err := db.c.Query(sqlQuery, myID, searchTerm)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Parse results
	var messages []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Content, &m.Kind, &m.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return messages, nil
}

func (db *appdbimpl) ForwardMessage(myID string, sourceMsgID string, destConvID string) error {

	// 1. Fetch Source Message Content
	// We join with participants to ensure the user (myID) is allowed to SEE the source message.
	var content string
	var kind string
	err := db.c.QueryRow(`
		SELECT m.content, m.kind 
		FROM messages m
		JOIN participants p ON m.conversation_id = p.conversation_id
		WHERE m.message_id = ? AND p.user_id = ?`,
		sourceMsgID, myID).Scan(&content, &kind)

	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("source message not found or access denied")
	}
	if err != nil {
		return err
	}

	// 2. Check Destination Permission
	// Ensure user (myID) is a participant in the destination chat
	var dummy int
	err = db.c.QueryRow(`
		SELECT 1 FROM participants 
		WHERE conversation_id = ? AND user_id = ?`,
		destConvID, myID).Scan(&dummy)

	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("destination conversation not found or access denied")
	}
	if err != nil {
		return err
	}

	// 3. Create the New Message (The Forward)
	// We generate a NEW ID for this forwarded message.
	newMessageID, err := GenerateNewID() // Use the public helper
	if err != nil {
		return err
	}

	_, err = db.c.Exec(`
		INSERT INTO messages (message_id, conversation_id, sender_id, content, kind, created_at) 
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`,
		newMessageID, destConvID, myID, content, kind)
	if err != nil {
		return err
	}

	// 4. Update Destination Conversation Metadata (Preview & Time)
	// Set last_message to "Forwarded: <content>" or just content
	preview := "Forwarded: " + content
	if kind == "image" {
		preview = "Forwarded: [Photo]"
	}
	if len(preview) > 50 {
		preview = preview[:47] + "..."
	}

	_, err = db.c.Exec(`
		UPDATE conversations 
		SET last_message = ?, updated_at = CURRENT_TIMESTAMP 
		WHERE conversation_id = ?`,
		preview, destConvID)

	return err
}

func (db *appdbimpl) DeleteMessage(messageID string, myID string, scope string) error {

	// 1. Check if message exists and who is the sender
	var senderID string
	err := db.c.QueryRow("SELECT sender_id FROM messages WHERE message_id = ?", messageID).Scan(&senderID)
	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("message not found")
	}
	if err != nil {
		return err
	}

	// 2. Security: Only the sender can delete (as per YAML description)
	if senderID != myID {
		return errors.New("forbidden: only sender can delete")
	}

	// 3. Handle Scopes
	if scope == "everyone" {
		// HARD DELETE: Remove from DB entirely
		// (Note: In a real app with Foreign Keys, you might need to delete from 'stars'/'deleted_messages' first
		//  or set ON DELETE CASCADE in your table creation. SQLite usually requires enabling Foreign Keys manually).

		// Safe approach: Delete references first (manual cascade)
		_, _ = db.c.Exec("DELETE FROM stars WHERE message_id = ?", messageID)
		_, _ = db.c.Exec("DELETE FROM deleted_messages WHERE message_id = ?", messageID)

		_, err = db.c.Exec("DELETE FROM messages WHERE message_id = ?", messageID)
		return err

	} else if scope == "me" {
		// SOFT DELETE: Add to deleted_messages table
		// IGNORE means "if already inserted, do nothing"
		_, err = db.c.Exec("INSERT OR IGNORE INTO deleted_messages (user_id, message_id) VALUES (?, ?)", myID, messageID)
		return err
	}

	return errors.New("invalid scope")
}
func (db *appdbimpl) GetAllStarredMessages(myID string) ([]Message, error) {

	// Query: Find messages M where (myID, m.id) exists in STARS table.
	// We don't filter by conversation_id here.
	query := `
		SELECT m.message_id, m.conversation_id, m.sender_id, m.content, m.kind, m.created_at
		FROM messages m
		JOIN stars s ON m.message_id = s.message_id
		WHERE s.user_id = ?
		ORDER BY m.created_at DESC`

	rows, err := db.c.Query(query, myID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Content, &m.Kind, &m.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return messages, nil
}
func (db *appdbimpl) StarMessage(messageID string, myID string) error {

	// 1. Security Check: Does the message exist and can I see it?
	// We join 'participants' to ensure the user is part of the conversation this message belongs to.
	var dummy int
	err := db.c.QueryRow(`
		SELECT 1 FROM messages m
		JOIN participants p ON m.conversation_id = p.conversation_id
		WHERE m.message_id = ? AND p.user_id = ?`,
		messageID, myID).Scan(&dummy)

	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("message not found or access denied")
	}
	if err != nil {
		return err
	}

	// 2. Star the message
	// INSERT OR IGNORE avoids errors if you star it twice
	_, err = db.c.Exec("INSERT OR IGNORE INTO stars (user_id, message_id) VALUES (?, ?)", myID, messageID)
	return err
}

func (db *appdbimpl) UnstarMessage(messageID string, myID string) error {

	// Execute Delete
	res, err := db.c.Exec("DELETE FROM stars WHERE user_id = ? AND message_id = ?", myID, messageID)
	if err != nil {
		return err
	}

	// Check if a row was actually deleted
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		// This covers both "Message Not Found" and "Message wasn't starred"
		return errors.New("star not found")
	}

	return nil
}

func (db *appdbimpl) ReactToMessage(messageID string, myID string, emoji string) error {

	// 1. Security Check: Does the message exist and can I see it?
	var dummy int
	err := db.c.QueryRow(`
		SELECT 1 FROM messages m
		JOIN participants p ON m.conversation_id = p.conversation_id
		WHERE m.message_id = ? AND p.user_id = ?`,
		messageID, myID).Scan(&dummy)

	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("message not found or access denied")
	}
	if err != nil {
		return err
	}

	// 2. Add/Update Reaction
	// "INSERT OR REPLACE" handles both adding and substituting
	_, err = db.c.Exec(`
		INSERT OR REPLACE INTO reactions (user_id, message_id, emoji) 
		VALUES (?, ?, ?)`,
		myID, messageID, emoji)

	return err
}
func (db *appdbimpl) UnreactToMessage(messageID string, myID string) error {

	// Execute Delete
	res, err := db.c.Exec("DELETE FROM reactions WHERE user_id = ? AND message_id = ?", myID, messageID)
	if err != nil {
		return err
	}

	// Check if we actually deleted anything
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("reaction not found")
	}

	return nil
}
