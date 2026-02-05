package database

import (
	"database/sql"
	"errors"
)

// GetOrCreateUser checks if a user exists.
// If yes, returns the ID.
// If no, creates a new user and returns the new ID.
func (db *appdbimpl) GetOrCreateUser(username string) (string, error) {
	var id string

	// 1) check if user already exists
	err := db.c.QueryRow("SELECT identifier FROM users WHERE username = ?", username).Scan(&id)

	// If yes, return existing ID
	if err == nil {
		return id, nil
	}

	// If the error is NOT "no rows found", it's a real error (like DB disconnected)
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}

	// 2) User does not exist, so let's create one.

	// Generate a new random ID (using the helper we put in database.go)
	newID, err := GenerateNewID()
	if err != nil {
		return "", err
	}

	// Insert the new user into the database
	_, err = db.c.Exec("INSERT INTO users (identifier, username) VALUES (?, ?)", newID, username)
	if err != nil {
		return "", err
	}

	// Return the new ID
	return newID, nil
}

// (db *appdbimpl) is the receiver of the function
func (db *appdbimpl) SearchUsers(query string) ([]User, error) {
	var rows *sql.Rows
	var err error

	// If the search query is empty, select ALL users
	if query == "" {
		rows, err = db.c.Query("SELECT identifier, username, photo_id FROM users ORDER BY username")
	} else {
		// If query exists, find exact matches
		rows, err = db.c.Query("SELECT identifier, username, photo_id FROM users WHERE username = ?", query)
	}

	if err != nil {
		return nil, err
	}
	// !! close rows to free up the connection
	defer func() { _ = rows.Close() }()

	// results
	var users []User
	for rows.Next() {
		var u User
		var photoID sql.NullString
		if err := rows.Scan(&u.Identifier, &u.Username, &photoID); err != nil {
			return nil, err
		}
		if photoID.Valid {
			u.PhotoID = photoID.String
		}
		users = append(users, u)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return users, nil
}

func (db *appdbimpl) GetUser(username string) (User, error) {
	var u User
	var photoID sql.NullString
	// select single user matching the username
	err := db.c.QueryRow("SELECT identifier, username, photo_id FROM users WHERE username = ?", username).Scan(&u.Identifier, &u.Username, &photoID)

	if photoID.Valid {
		u.PhotoID = photoID.String
	}
	// if error is "no rows found", user doesnt exist
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, err
	}
	// if its another error (lost connection...) return it
	if err != nil {
		return User{}, err
	}
	return u, nil
}

func (db *appdbimpl) SetUsername(id string, newName string) error {
	// 1. Try to update the username
	_, err := db.c.Exec("UPDATE users SET username = ? WHERE identifier = ?", newName, id)

	// 2. Return the error (if any)
	return err
}

func (db *appdbimpl) SetUserPhoto(id string, photoID string) error {
	// Update the users table to point to the new photo
	_, err := db.c.Exec("UPDATE users SET photo_id = ? WHERE identifier = ?", photoID, id)
	return err
}
