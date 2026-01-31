package api

import (
	"net/http"
)

// Handler returns an instance of httprouter.Router that handle APIs registered here
func (rt *_router) Handler() http.Handler {

	// LOGIN
	rt.router.POST("/session", rt.wrap(rt.doLogin))

	// USERS
	rt.router.GET("/users", rt.wrap(rt.searchUsers))
	rt.router.GET("/users/:username", rt.wrap(rt.getUserProfile))
	rt.router.PUT("/user/name", rt.wrap(rt.setMyUserName))
	rt.router.PUT("/user/photo", rt.wrap(rt.setMyPhoto))

	// CONVERSATIONS
	rt.router.GET("/conversations", rt.wrap(rt.getMyConversations))
	rt.router.POST("/conversations", rt.wrap(rt.startConversation))
	rt.router.GET("/conversations/:conversationId", rt.wrap(rt.getConversation))
	rt.router.DELETE("/conversations/:conversationId", rt.wrap(rt.deleteConversation))

	// MESSAGES
	rt.router.POST("/conversations/:conversationId/messages", rt.wrap(rt.sendMessage))
	rt.router.POST("/conversations/:conversationId/photos", rt.wrap(rt.sendPhotoMessage))
	rt.router.GET("/conversations/:conversationId/photos", rt.wrap(rt.getConvPhotos))
	rt.router.GET("/conversations/:conversationId/messages/starred", rt.wrap(rt.getConvStarredMessages))
	rt.router.GET("/messages", rt.wrap(rt.searchMessages))
	rt.router.POST("/messages/forwarded", rt.wrap(rt.forwardMessage))
	rt.router.DELETE("/messages/:messageId", rt.wrap(rt.deleteMessage))

	// STARRED & REACTIONS
	rt.router.GET("/messages/starred", rt.wrap(rt.getAllStarredMessages))
	rt.router.PUT("/messages/:messageId/star", rt.wrap(rt.starMessage))
	rt.router.DELETE("/messages/:messageId/star", rt.wrap(rt.unstarMessage))
	rt.router.PUT("/messages/:messageId/reactions/me", rt.wrap(rt.commentMessage))
	rt.router.DELETE("/messages/:messageId/reactions/me", rt.wrap(rt.uncommentMessage))

	// GROUPS
	rt.router.POST("/groups", rt.wrap(rt.createGroup))
	rt.router.GET("/groups/:groupId", rt.wrap(rt.getGroupDetails))
	rt.router.DELETE("/user/groups/:groupId", rt.wrap(rt.leaveGroup))
	rt.router.POST("/groups/:groupId/participants", rt.wrap(rt.addToGroup))
	rt.router.PUT("/groups/:groupId/name", rt.wrap(rt.setGroupName))
	rt.router.PUT("/groups/:groupId/photo", rt.wrap(rt.setGroupPhoto))

	return rt.router
}
