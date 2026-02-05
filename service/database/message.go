package database

import (
	"database/sql"
	"errors"
	"strings"
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
	// 2. Fetch "Oldest" Last Seen among OTHER participants
	// If everyone else has seen it (last_seen > msg.created_at), then it is "read".
	var othersLastSeen string
	var stragglerName string // DEBUG
	err = db.c.QueryRow(`
        SELECT MIN(p.last_seen), (SELECT username FROM users WHERE identifier = p.user_id)
        FROM participants p
        WHERE p.conversation_id = ? AND p.user_id != ?`, conversationID, myID).Scan(&othersLastSeen, &stragglerName)

	// If err (e.g. no other participants?), treat as empty (not read)
	if err != nil {
		othersLastSeen = ""
	}

	// 3. Fetch Messages (Updated with Status calculation logic, Reply details, and IsStarred)
	rows, err := db.c.Query(`
		SELECT 
            m.message_id, m.conversation_id, m.sender_id, u.username, m.content, m.photo_source, m.kind, m.created_at, m.is_forwarded,
            m.reply_to, r.sender_id, ru.username, r.content, r.photo_source,
            CASE WHEN s.message_id IS NOT NULL THEN 1 ELSE 0 END AS is_starred
		FROM messages m
		JOIN participants p ON m.conversation_id = p.conversation_id
        JOIN users u ON m.sender_id = u.identifier
        LEFT JOIN deleted_messages d ON m.message_id = d.message_id AND d.user_id = ?
        LEFT JOIN messages r ON m.reply_to = r.message_id
        LEFT JOIN users ru ON r.sender_id = ru.identifier
        LEFT JOIN stars s ON m.message_id = s.message_id AND s.user_id = ?
		WHERE m.conversation_id = ? 
          AND p.user_id = ? 
          AND d.message_id IS NULL  
		ORDER BY m.created_at ASC`,
		myID, myID, conversationID, myID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		// Need to scan nullable photo_source request params
		var photoSource sql.NullString
		var replyTo sql.NullString
		var replySenderID sql.NullString
		var replySenderName sql.NullString
		var replyContent sql.NullString
		var replyPhotoSource sql.NullString
		var isStarredInt int

		if err := rows.Scan(
			&m.ID, &m.ConversationID, &m.SenderID, &m.SenderName, &m.Content, &photoSource, &m.Kind, &m.CreatedAt, &m.IsForwarded,
			&replyTo, &replySenderID, &replySenderName, &replyContent, &replyPhotoSource, &isStarredInt,
		); err != nil {
			return nil, err
		}
		m.IsStarred = (isStarredInt > 0)
		if photoSource.Valid {
			m.PhotoSource = photoSource.String
		}
		if replyTo.Valid {
			m.ReplyTo = replyTo.String
			m.ReplyToSender = replySenderName.String
			// Determine preview text for reply
			if replyContent.String != "" {
				m.ReplyToText = replyContent.String
			} else if replyPhotoSource.Valid {
				m.ReplyToText = "[Photo]"
			}
		}

		// Normalize CreatedAt for comparison (Remove 'T' and 'Z' to match SQL format)
		cmpTime := m.CreatedAt
		cmpTime = strings.Replace(cmpTime, "T", " ", 1)
		cmpTime = strings.TrimSuffix(cmpTime, "Z")

		// Calculate Status
		if othersLastSeen != "" && cmpTime <= othersLastSeen {
			m.Status = "read"
		} else {
			m.Status = "received"
		}

		messages = append(messages, m)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	// 4. Fetch Reactions for these messages
	// We do this in a separate query for simplicity (avoiding complex group concat logic in SQL)
	// We want: emoji, count, and if *I* reacted.
	// We can loop through messages or do one big query with IN clause.
	// Given the scale, looping with a prepared statement is "okay" but one query is better.
	// Let's iterate and populate.
	for i := range messages {
		// Subquery for each message (simplest to implement reliably)
		// SELECT emoji, COUNT(*), MAX(CASE WHEN user_id = ? THEN 1 ELSE 0 END) FROM reactions WHERE message_id = ? GROUP BY emoji
		rRows, err := db.c.Query(`
			SELECT emoji, COUNT(*), MAX(CASE WHEN user_id = ? THEN 1 ELSE 0 END) 
			FROM reactions 
			WHERE message_id = ? 
			GROUP BY emoji`, myID, messages[i].ID)
		if err != nil {
			return nil, err
		}

		// Initialize as empty slice so it returns [] instead of null in JSON if empty (optional preference)
		var reactions = []Reaction{}

		for rRows.Next() {
			var r Reaction
			var isMeVal sql.NullInt64 // Use NullInt64 to be safe against NULL

			if err := rRows.Scan(&r.Emoji, &r.Count, &isMeVal); err != nil {
				rRows.Close()
				return nil, err
			}
			r.IsMe = (isMeVal.Valid && isMeVal.Int64 > 0)
			reactions = append(reactions, r)
		}
		rRows.Close()

		messages[i].Reactions = reactions
	}

	return messages, nil
}

// check if you are a participant, generate a new Message ID,
// insert the message and update the parent conversation (s.t."Last Message" preview changes).
func (db *appdbimpl) SendMessage(conversationID string, senderID string, text string, replyTo string) (Message, error) {

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
		INSERT INTO messages (message_id, conversation_id, sender_id, content, kind, reply_to, created_at) 
		VALUES (?, ?, ?, ?, 'text', ?, CURRENT_TIMESTAMP)`,
		messageID, conversationID, senderID, text, sql.NullString{String: replyTo, Valid: replyTo != ""})
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
		// fmt.Printf("[DEBUG] SendMessage UPDATE failed for conv %s: %v\n", conversationID, err)
		return Message{}, err
	}
	// rowsAff, _ := res.RowsAffected()
	// fmt.Printf("[DEBUG] SendMessage UPDATE success for conv %s. Rows affected: %d. Preview: '%s'\n", conversationID, rowsAff, preview)

	// 5. Return the full object

	// We need to fetch the 'created_at' time that the DB just generated
	var createdAt string
	err = db.c.QueryRow("SELECT created_at FROM messages WHERE message_id = ?", messageID).Scan(&createdAt)
	if err != nil {
		return Message{}, err
	}

	// Fetch my Username to return in SenderName
	var myUsername string
	err = db.c.QueryRow("SELECT username FROM users WHERE identifier = ?", senderID).Scan(&myUsername)
	if err != nil {
		return Message{}, err
	}

	// Fetch ReplyTo details if applicable?
	// Usually frontend has it, but consistent return is good.
	// Here we just return what we inserted.
	// If replyTo was valid, we assume frontend knows who it was.
	// Ideally we should look it up if we want to be 100% correct, but simple return is okay for immediate feedback.

	return Message{
		ID:             messageID,
		ConversationID: conversationID,
		SenderID:       senderID,
		SenderName:     myUsername,
		Content:        text,
		Kind:           "text",
		ReplyTo:        replyTo,
		CreatedAt:      createdAt,
	}, nil
}

func (db *appdbimpl) SendPhotoMessage(conversationID string, senderID string, photoID string, caption string, replyTo string) error {

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
	messageID, err := GenerateNewID()
	if err != nil {
		return err
	}

	// 3. Insert Message
	// kind is 'image'. Content is caption. PhotoSource is photoID.
	_, err = db.c.Exec(`
		INSERT INTO messages (message_id, conversation_id, sender_id, content, photo_source, kind, reply_to, created_at) 
		VALUES (?, ?, ?, ?, ?, 'image', ?, CURRENT_TIMESTAMP)`,
		messageID, conversationID, senderID, caption, photoID, sql.NullString{String: replyTo, Valid: replyTo != ""})
	if err != nil {
		return err
	}

	// 4. Update Conversation Metadata
	// For images, the "last message" preview is usually "[Image]" or "Photo"
	preview := "[Photo]"
	if caption != "" {
		preview = "[Photo] " + caption
	}
	_, err = db.c.Exec(`
		UPDATE conversations 
		SET last_message = ?, updated_at = CURRENT_TIMESTAMP 
		WHERE conversation_id = ?`,
		preview, conversationID)

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
	// We only select 'photo_source' where kind='image'
	rows, err := db.c.Query(`
		SELECT photo_source 
		FROM messages 
		WHERE conversation_id = ? AND kind = 'image' AND photo_source IS NOT NULL
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
		SELECT m.message_id, m.conversation_id, m.sender_id, u.username, m.content, m.kind, m.created_at
		FROM messages m
		JOIN stars s ON m.message_id = s.message_id
        JOIN users u ON m.sender_id = u.identifier
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
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.SenderName, &m.Content, &m.Kind, &m.CreatedAt); err != nil {
			return nil, err
		}
		m.IsStarred = true
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
		SELECT m.message_id, m.conversation_id, m.sender_id, u.username, m.content, m.kind, m.created_at,
               CASE WHEN s.message_id IS NOT NULL THEN 1 ELSE 0 END
		FROM messages m
		JOIN participants p ON m.conversation_id = p.conversation_id
		LEFT JOIN stars s ON m.message_id = s.message_id AND s.user_id = ?
        JOIN users u ON m.sender_id = u.identifier
		WHERE p.user_id = ? 
		  AND m.kind = 'text'
		  AND m.content LIKE ?
		ORDER BY m.created_at DESC
	`

	// Add wildcards for partial matching
	searchTerm := "%" + query + "%"

	rows, err := db.c.Query(sqlQuery, myID, myID, searchTerm)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Parse results
	var messages []Message
	for rows.Next() {
		var m Message
		var isStarredInt int
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.SenderName, &m.Content, &m.Kind, &m.CreatedAt, &isStarredInt); err != nil {
			return nil, err
		}
		m.IsStarred = (isStarredInt > 0)
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
	var photoSource sql.NullString // Handle potential NULL photo_source

	err := db.c.QueryRow(`
		SELECT m.content, m.kind, m.photo_source
		FROM messages m
		JOIN participants p ON m.conversation_id = p.conversation_id
		WHERE m.message_id = ? AND p.user_id = ?`,
		sourceMsgID, myID).Scan(&content, &kind, &photoSource)

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
		INSERT INTO messages (message_id, conversation_id, sender_id, content, kind, photo_source, is_forwarded, created_at) 
		VALUES (?, ?, ?, ?, ?, ?, TRUE, CURRENT_TIMESTAMP)`,
		newMessageID, destConvID, myID, content, kind, photoSource)
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
	switch scope {
	case "everyone":
		// HARD DELETE: Remove from DB entirely
		// (Note: In a real app with Foreign Keys, you might need to delete from 'stars'/'deleted_messages' first
		//  or set ON DELETE CASCADE in your table creation. SQLite usually requires enabling Foreign Keys manually).

		// Safe approach: Delete references first (manual cascade)
		_, _ = db.c.Exec("DELETE FROM stars WHERE message_id = ?", messageID)
		_, _ = db.c.Exec("DELETE FROM deleted_messages WHERE message_id = ?", messageID)

		_, err = db.c.Exec("DELETE FROM messages WHERE message_id = ?", messageID)
		return err

	case "me":
		// SOFT DELETE: Add to deleted_messages table
		// IGNORE means "if already inserted, do nothing"
		_, err = db.c.Exec("INSERT OR IGNORE INTO deleted_messages (user_id, message_id) VALUES (?, ?)", myID, messageID)
		return err

	default:
		return errors.New("invalid scope")
	}
}
func (db *appdbimpl) GetAllStarredMessages(myID string) ([]Message, error) {

	// Query: Find messages M where (myID, m.id) exists in STARS table.
	// We don't filter by conversation_id here.
	// Also join users to get sender name
	query := `
		SELECT m.message_id, m.conversation_id, m.sender_id, u.username, m.content, m.kind, m.created_at
		FROM messages m
		JOIN stars s ON m.message_id = s.message_id
        JOIN users u ON m.sender_id = u.identifier
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
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.SenderName, &m.Content, &m.Kind, &m.CreatedAt); err != nil {
			return nil, err
		}
		// Fix Time string for browser compatibility
		m.CreatedAt = strings.Replace(m.CreatedAt, " ", "T", 1) + "Z"

		m.IsStarred = true // By definition because they are in this list
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
