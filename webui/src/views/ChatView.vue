<script>
import MessageItem from '../components/MessageItem.vue'
import ForwardModal from '../components/ForwardModal.vue'
import GroupSettingsModal from '../components/GroupSettingsModal.vue'
import ImageViewerModal from '../components/ImageViewerModal.vue'
import DeleteMessageModal from '../components/DeleteMessageModal.vue'
import ReactionListModal from '../components/ReactionListModal.vue'

export default {
    components: { MessageItem, ForwardModal, GroupSettingsModal, ImageViewerModal, DeleteMessageModal, ReactionListModal },
	data() {
		return {
			messages: [],
			newMessage: "",
            loading: false,
            conversationName: "",
            selectedFile: null,
            replyingTo: null, // Message object being replied to

            showForwardModal: false,
            showSettingsModal: false,
            
            // Deletion
            messageToDelete: null,
            showDeleteModal: false,

            selectedImage: null, // URL of image to view
            messageToForward: null, // Single message (legacy/direct click)
            selectionMode: false,
            selectedMessages: [], // Array of message IDs

            // Search
            isSearching: false,
            searchQuery: "",

            // Reaction Details
            reactionMessageId: null,
            showReactionModal: false,
		}
	},
    computed: {
        filteredMessages() {
            if (!this.searchQuery) return this.messages;
            const lowerQuery = this.searchQuery.toLowerCase();
            return this.messages.filter(m => 
                (m.textContent && m.textContent.toLowerCase().includes(lowerQuery)) ||
                (m.senderName && m.senderName.toLowerCase().includes(lowerQuery))
            );
        },
        conversationId() {
            return this.$route.params.conversationId;
        },
        conversationTitle() {
            if (this.conversationName) return this.conversationName;
            return "Chat";
        }
    },
    watch: {
        '$route.params.conversationId': {
            handler: 'refresh',
            immediate: true
        }
    },
	methods: {
        async refresh() {
            if (!this.conversationId) return;
            // this.loading = true; // Don't show spinner on poll
            try {
                // Fetch messages
                let response = await this.$axios.get(`/conversations/${this.conversationId}/messages`);
                this.messages = response.data;
                
                // Get Title from Root
                if (this.$root.conversations) {
                    const conv = this.$root.conversations.find(c => c.conversationId === this.conversationId);
                    if (conv) {
                        this.conversationName = conv.conversationName;
                    } else {
                         // Fallback: Use "Other Person's Name" from messages for DM
                         const other = this.messages.find(m => m.senderName !== localStorage.getItem("username"));
                         if (other) {
                             this.conversationName = other.senderName;
                         }
                    }
                }
                
                // Mark as Seen
                await this.$axios.put(`/conversations/${this.conversationId}/seen`);

                // Scroll to bottom if we are near bottom or first load? (Simplified: always scroll for now or relying on polling not jumping)
                // this.$nextTick(() => { this.scrollToBottom(); });
            } catch (e) {
                console.error(e);
            }
            this.loading = false;
        },
        
        onFileSelected(event) {
            const file = event.target.files[0];
            if (file) {
                this.selectedFile = file;
            }
            // Clear input value so same file can be selected again if needed
            event.target.value = "";
        },
        
        clearFile() {
            this.selectedFile = null;
        },

        // Reply Logic
        setReply(message) {
            this.replyingTo = message;
            this.$nextTick(() => {
                document.querySelector('input[type="text"]').focus();
            });
        },
        clearReply() {
            this.replyingTo = null;
        },

        // Selection Logic
        toggleSelectionMode() {
            this.selectionMode = !this.selectionMode;
            this.selectedMessages = [];
        },
        toggleSelection(message) {
            if (!this.selectionMode) return;
            const index = this.selectedMessages.indexOf(message.messageId);
            if (index === -1) {
                this.selectedMessages.push(message.messageId);
            } else {
                this.selectedMessages.splice(index, 1);
            }
        },

        // Forward Logic
        handleForward(message) {
            // If passed a specific message (from button click), use that.
            // If null/undefined, assume we are forwarding SELECTION.
            if (message) {
                this.messageToForward = message;
            }
            this.showForwardModal = true;
        },
        async onForwardConfirm(destConvId) {
            // Determine source IDs
            let sourceIds = [];
            if (this.messageToForward) {
                sourceIds = [this.messageToForward.messageId];
            } else if (this.selectionMode && this.selectedMessages.length > 0) {
                sourceIds = this.selectedMessages;
            }

            if (sourceIds.length === 0) return;

            try {
                await this.$axios.post(`/conversations/${destConvId}/forward`, {
                    sourceMessageIds: sourceIds
                });
                // If forwarding to CURRENT chat, refresh; otherwise, maybe show toast?
                if (destConvId === this.conversationId) {
                    this.refresh();
                }
                this.showForwardModal = false;
                this.messageToForward = null;
                // Exit selection mode on success
                if (this.selectionMode) {
                    this.toggleSelectionMode();
                }
            } catch (e) {
                alert("Error forwarding message: " + (e.response?.data?.message || e.toString()));
            }
        },

        // Group Settings Logic
        handleNameUpdate(newName) {
            this.conversationName = newName;
            // Also notify root to update sidebar list immediately? 
            // The sidebar polls independently, but we can try to force valid updates if we had a global bus.
            // For now, local update is good feedback.
        },
        handleLeave() {
            this.showSettingsModal = false;
            this.$router.push('/'); // Go back to home
        },

        confirmDelete(msg) {
            this.messageToDelete = msg;
            this.showDeleteModal = true;
        },

        async executeDelete(messageId, scope) {
            try {
                await this.$axios.delete(`/messages/${messageId}`, {
                    params: { scope: scope }
                });
                // Remove from local list immediately (optimistic UI)
                this.messages = this.messages.filter(m => m.messageId !== messageId);
            } catch (e) {
                alert("Error deleting message: " + (e.response?.data?.error || e.toString()));
            }
            this.showDeleteModal = false;
            this.messageToDelete = null;
        },


        
        // Reaction Logic
        async toggleReaction(message, emoji) {
            // Check if I already reacted with ALL emojis? No, just check if I reacted with THIS emoji.
            // But the backend API is PUT /reactions/me (which might imply one reaction per message? Or one per emoji?)
            // The DB schema has PK (user_id, message_id, emoji) -> wait, looking at schema provided earlier...
            // Step Id: 1251 shows: `PRIMARY KEY (user_id, message_id)` implies ONE EMOJI PER USER PER MESSAGE!
            // Wait, let me double check the database schema in Step 1251.
            // Line 202: PRIMARY KEY (user_id, message_id).
            // Line 201: emoji TEXT NOT NULL.
            // YES. A user can only have ONE reaction per message.
            
            // So if I click a DIFFERENT emoji, it replaces.
            // If I click the SAME emoji, it should remove it (toggle).
            
            // Find my current reaction
            const myCurrentReaction = message.reactions?.find(r => r.isMe);
            const isRemoving = myCurrentReaction && myCurrentReaction.emoji === emoji;
            
            try {
                if (isRemoving) {
                    await this.$axios.delete(`/reactions/${message.messageId}`);
                } else {
                    await this.$axios.put(`/reactions/${message.messageId}`, { emoticon: emoji });
                }
                // Refresh to see update
                this.refresh(); 
            } catch(e) {
                console.error("Reaction error", e);
            }
        },

        async handleSend() {
            if (this.selectedFile) {
                // Send as Photo Message
                await this.sendPhoto();
            } else {
                // Send as Text Message
                await this.sendText();
            }
        },

        async sendText() {
            if (!this.newMessage.trim()) return;
            try {
                const payload = {
                    textContent: this.newMessage
                };
                if (this.replyingTo) {
                    payload.replyTo = this.replyingTo.messageId;
                }

                await this.$axios.post(`/conversations/${this.conversationId}/messages`, payload);
                this.newMessage = "";
                this.replyingTo = null; // Clear reply after send
                this.refresh();
                this.$nextTick(() => this.scrollToBottom());
            } catch (e) {
                alert("Error sending message: " + e.toString());
            }
        },

        async sendPhoto() {
            if (!this.selectedFile) return;

            let formData = new FormData();
            formData.append("photo", this.selectedFile);
            
            // Append current text as caption (optional)
            if (this.newMessage.trim()) {
                formData.append("caption", this.newMessage.trim());
            }
            if (this.replyingTo) {
                formData.append("replyTo", this.replyingTo.messageId);
            }

            this.loading = true;
            try {
                await this.$axios.post(`/conversations/${this.conversationId}/photos`, formData, {
                    headers: {
                        'Content-Type': 'multipart/form-data'
                    }
                });
                this.newMessage = ""; 
                this.selectedFile = null; // Clear file after send
                this.replyingTo = null;
                this.refresh();
                this.$nextTick(() => this.scrollToBottom());
            } catch (e) {
                alert("Error sending photo: " + (e.response?.data?.message || e.toString()));
            }
            this.loading = false;
        },
        
        scrollToBottom() {
            const container = this.$refs.messageContainer;
            if (container) {
                container.scrollTop = container.scrollHeight;
            }
        },

        async toggleStar(message) {
            const isStarred = message.isStarred;
            console.log("Toggling star for", message.messageId, "current:", isStarred);
            // Optimistic update
            message.isStarred = !isStarred;

            try {
                if (isStarred) {
                    await this.$axios.delete(`/messages/${message.messageId}/star`);
                } else {
                    await this.$axios.put(`/messages/${message.messageId}/star`);
                }
            } catch (e) {
                console.error("Failed to toggle star", e);
                // Revert
                message.isStarred = isStarred;
                alert("Failed to star message: " + (e.response?.data?.error || e.toString()));
            }
        }
	},
    mounted() {
        // Poll for new messages
        this.pollInterval = setInterval(() => {
            this.refresh();
        }, 3000);
    },
    beforeUnmount() {
        clearInterval(this.pollInterval);
    }
}
</script>


<template>
    <div class="d-flex flex-column h-100 position-relative">
        <!-- Header -->
        <div class="p-3 bg-surface d-flex justify-content-between align-items-center shadow-sm" style="z-index: 10;">
            <div class="d-flex align-items-center gap-2">
                 <!-- Back button for mobile -->
                 <button class="btn btn-sm btn-light d-md-none rounded-circle" @click="$router.push('/')">
                    <svg class="feather"><use href="/feather-sprite-v4.29.0.svg#arrow-left"/></svg>
                </button>
                <h5 class="m-0 text-truncate fw-bold text-body" style="max-width: 300px;">{{ conversationTitle }}</h5>
            </div>
            
            <div class="d-flex gap-2">
                <!-- Selection Mode Actions -->
                <div v-if="selectionMode" class="d-flex gap-2 animate__animated animate__fadeIn">
                    <button class="btn btn-sm btn-primary rounded-pill px-3" @click="handleForward(null)" :disabled="selectedMessages.length === 0">
                        Forward ({{ selectedMessages.length }})
                    </button>
                    <button class="btn btn-sm btn-light rounded-pill px-3" @click="toggleSelectionMode">Cancel</button>
                </div>
                
                <!-- Default Actions -->
                <div v-else class="d-flex gap-2 align-items-center">
                    <!-- Search Bar -->
                    <div v-if="isSearching" class="animate__animated animate__fadeInRight d-flex gap-1" style="width: 200px;">
                        <input type="text" v-model="searchQuery" class="form-control form-control-sm rounded-pill" placeholder="Search..." auto-focus>
                        <button class="btn btn-sm btn-light rounded-circle" @click="isSearching = false; searchQuery = ''">
                            <svg class="feather"><use href="/feather-sprite-v4.29.0.svg#x"/></svg>
                        </button>
                    </div>
                    <button v-else class="btn btn-sm btn-light rounded-circle text-muted" @click="isSearching = true">
                        <svg class="feather"><use href="/feather-sprite-v4.29.0.svg#search"/></svg>
                    </button>

                    <button class="btn btn-sm btn-light rounded-circle text-muted" @click="toggleSelectionMode" title="Select Messages">
                        <svg class="feather"><use href="/feather-sprite-v4.29.0.svg#check-square"/></svg>
                    </button>
                    <button class="btn btn-sm btn-light rounded-circle text-muted" @click="showSettingsModal = true">
                        <svg class="feather"><use href="/feather-sprite-v4.29.0.svg#info"/></svg>
                    </button>
                </div>
            </div>
        </div>

        <!-- Messages -->
        <div class="flex-grow-1 overflow-auto p-3" style="background-color: var(--app-bg);" ref="messageContainer">
            <div v-if="loading && messages.length === 0" class="text-center mt-5">
                <LoadingSpinner />
            </div>
            <div v-else-if="messages.length === 0" class="text-center text-muted mt-5">
                <div class="p-4 rounded-4 bg-surface d-inline-block shadow-sm">
                    <span class="d-block mb-2 fs-4">👋</span>
                    No messages yet. Say hello!
                </div>
            </div>
            <div v-else class="d-flex flex-column gap-2">
                <MessageItem 
                    v-for="msg in filteredMessages" 
                    :key="msg.messageId" 
                    :message="msg" 
                    :is-me="msg.senderName === $root.username" 
                    :selectable="selectionMode"
                    :selected="selectedMessages.includes(msg.messageId)"
                    @reply="setReply" 
                    @forward="handleForward" 
                    @toggle-selection="toggleSelection"
                    @react="toggleReaction"
                    @toggle-star="toggleStar"
                    @view-image="(src) => selectedImage = src"
                    @delete="confirmDelete"
                    @view-reactions="(msg) => { reactionMessageId = msg.messageId; showReactionModal = true; }"
                />
            </div>
        </div>

        <!-- Input Area -->
        <div class="p-3 bg-surface">
            <!-- Reply Preview -->
            <div v-if="replyingTo" class="mb-2 p-2 border-start border-4 border-primary bg-body-secondary rounded-3 d-flex justify-content-between align-items-center">
                <div class="small ps-2">
                    <div class="fw-bold text-primary">Replying to {{ replyingTo.senderName }}</div>
                    <div class="text-truncate text-muted" style="max-width: 300px;">
                        {{ replyingTo.textContent || (replyingTo.photoSource ? '[Photo]' : '') }}
                    </div>
                </div>
                <button type="button" class="btn-close small" aria-label="Close" @click="clearReply"></button>
            </div>

            <!-- File Preview -->
            <div v-if="selectedFile" class="mb-2 position-relative d-inline-block">
                <div class="border rounded-3 p-2 d-flex align-items-center gap-2 bg-light shadow-sm">
                    <span class="small text-truncate fw-bold" style="max-width: 200px;">{{ selectedFile.name }}</span>
                    <button type="button" class="btn-close small" aria-label="Close" @click="clearFile"></button>
                </div>
            </div>

            <form @submit.prevent="handleSend" class="d-flex gap-2 align-items-center bg-surface p-1 rounded-pill border" style="border-color: var(--input-bg) !important;">
                <!-- File Button -->
                 <button type="button" class="btn btn-link text-secondary rounded-circle" @click="$refs.fileInput.click()" title="Attach Photo" style="width: 40px; height: 40px;">
                    <svg class="feather"><use href="/feather-sprite-v4.29.0.svg#image"/></svg>
                </button>
                <input type="file" ref="fileInput" class="d-none" accept="image/*" @change="onFileSelected">
                
                <input type="text" v-model="newMessage" class="form-control border-0 bg-transparent shadow-none" placeholder="Type a message..." :required="!selectedFile" style="font-size: 0.95rem;">
                
                <button type="submit" class="btn btn-primary rounded-circle d-flex align-items-center justify-content-center p-0 shadow-sm" style="width: 40px; height: 40px; min-width: 40px;">
                    <svg class="feather" style="width: 18px; height: 18px;"><use href="/feather-sprite-v4.29.0.svg#send"/></svg>
                </button>
            </form>
        </div>

        <!-- Modals -->
        <ForwardModal 
            v-if="showForwardModal" 
            :conversations="$root.conversations" 
            :current-conversation-id="conversationId"
            @close="showForwardModal = false" 
            @confirm="onForwardConfirm" 
        />

        <GroupSettingsModal
            v-if="showSettingsModal"
            :conversation-id="conversationId"
            :initial-name="conversationName"
            :conversation="$root.conversations.find(c => c.conversationId === conversationId)"
            :is-group="!!($root.conversations.find(c => c.conversationId === conversationId)?.isGroup)"
            @close="showSettingsModal = false"
            @update-name="handleNameUpdate"
            @leave="handleLeave"
        />

        <ImageViewerModal
            v-if="selectedImage"
            :image-src="selectedImage"
            @close="selectedImage = null"
        />

        <DeleteMessageModal
            v-if="showDeleteModal"
            :message="messageToDelete"
            :is-me="messageToDelete?.senderName === $root.username"
            @close="showDeleteModal = false"
            @confirm="executeDelete"
        />

        <ReactionListModal
            v-if="showReactionModal"
            :message-id="reactionMessageId"
            @close="showReactionModal = false"
        />
	</div>
</template>

<style scoped>
</style>
