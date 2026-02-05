package database

import "fmt"

func (db *appdbimpl) GetMessageReactions(messageID string) ([]ReactionDetail, error) {
	// Query to fetch username and emoji for a specific message
	// We don't necessarily need to check permissions here if the API handler validates the user is in the conversation first.
	// But good practice: The API handler calls 'GetMessageReactions', it usually implies the user ALREADY loaded the message.
	// However, strictly speaking, we should perhaps check if the requestor has access?
	// The API handler will likely only pass the messageID.
	// Let's assume the API handler handles the "Can I see this message?" check or we just return public info for that message.
	// Actually, checking permissions inside DB is safer. But we need 'myID' for that.
	// The interface I defined only takes 'messageID'.
	// Given the scope, let's just fetch the data. The API layer should ensure the user is authorized to view the message (by checking conversation membership).

	rows, err := db.c.Query(`
		SELECT u.username, r.emoji
		FROM reactions r
		JOIN users u ON r.user_id = u.identifier
		WHERE r.message_id = ?
		ORDER BY u.username ASC`, messageID)
	if err != nil {
		return nil, fmt.Errorf("error searching reactions: %w", err)
	}
	defer rows.Close()

	var details []ReactionDetail
	for rows.Next() {
		var r ReactionDetail
		if err := rows.Scan(&r.Username, &r.Emoji); err != nil {
			return nil, fmt.Errorf("error scanning reaction: %w", err)
		}
		details = append(details, r)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return details, nil
}
