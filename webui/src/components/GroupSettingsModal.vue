<script>
import ImageViewerModal from './ImageViewerModal.vue'

export default {
    components: { ImageViewerModal },
    props: ['conversationId', 'initialName', 'conversation', 'isGroup'],
    emits: ['close', 'update-name', 'leave'],
    data() {
        return {
            loading: true,
            groupStub: {
                groupName: this.initialName || "Group",
                members: [],
                groupPhoto: null
            },
            newName: "",
            isEditingName: false,
            photoFile: null,
            uploadingPhoto: false,
            isAddingMember: false,
            newMemberQuery: "",
            searchResults: [],
            
            // New for Starred & Photos
            activeTab: 'members', // 'members', 'starred', 'photos'
            starredMessages: [],
            starredLoading: false,
            photos: [],
            photosLoading: false,
            selectedImage: null,
            realIsGroup: this.isGroup // Initialize with prop, but allow override
        }
    },
    async mounted() {
        // ALWAYS try to fetch group details. If it succeeds, it IS a group.
        // If it fails (404), it is a DM. This fixes the issue where the list might have wrong isGroup flag.
        this.loading = true;
        try {
            await this.fetchDetails();
            this.realIsGroup = true;
            this.activeTab = 'members';
        } catch (e) {
            // It's a DM (404 expected), so we treat it as such.
            this.realIsGroup = false;
            this.activeTab = 'photos';
            
            // FALLBACK STRATEGY: Directly access root data to bypass prop issues
            // Since Sidebar works, $root.conversations MUST have the data.
            let directConv = null;
            if (this.$root && this.$root.conversations) {
                directConv = this.$root.conversations.find(c => c.conversationId === this.conversationId);
            }
            
            if (directConv) {
                this.groupStub.groupName = directConv.conversationName || 'Chat Info';
                this.groupStub.groupPhoto = directConv.conversationPhoto;
            } else {
                // Last resort: props
                this.groupStub.groupName = (this.initialName && this.initialName !== 'Group') ? this.initialName : 'Chat Info';
                this.groupStub.groupPhoto = this.conversation ? this.conversation.conversationPhoto : null;
            }
        }
        
        // Initial fetch for tabs
        if (this.activeTab === 'starred') {
            this.fetchStarred();
        } else if (this.activeTab === 'photos') {
            this.fetchPhotos();
        }
        this.loading = false;
    },
    watch: {
        activeTab(newTab) {
            if (newTab === 'starred') {
                this.fetchStarred();
            } else if (newTab === 'photos') {
                this.fetchPhotos();
            }
        },
        initialName: {
             handler(newVal) {
                 // If we are treating as DM (realIsGroup is false), update the name dynamically
                 if (!this.realIsGroup) {
                     this.groupStub.groupName = (newVal && newVal !== 'Group') ? newVal : 'Chat Info';
                 }
             },
             immediate: true
        },
        conversation: {
            handler(newVal) {
                console.log("GSM Watch: conversation updated:", newVal);
                if (newVal && !this.realIsGroup) {
                     this.groupStub.groupPhoto = newVal.conversationPhoto;
                     // Also re-apply name logic just in case
                     if (this.initialName && this.initialName !== 'Group') {
                         this.groupStub.groupName = this.initialName;
                     }
                }
            },
            immediate: true
        }
    },
    methods: {
        async fetchPhotos() {
            this.photosLoading = true;
            try {
                const response = await this.$axios.get(`/conversations/${this.conversationId}/photos`);
                console.log("FETCH PHOTOS RESPONSE:", response.data);
                this.photos = response.data || [];
            } catch (e) {
                console.error("Failed to fetch photos", e);
            }
            this.photosLoading = false;
        },
        async fetchStarred() {
             this.starredLoading = true;
             try {
                let response = await this.$axios.get(`/conversations/${this.conversationId}/messages/starred`); // Verify API path
                // Wait, the API path in api-handler.go is:
                // rt.router.GET("/conversations/:conversationId/messages/starred", rt.wrap(rt.getConvStarredMessages))
                // Matches!
                this.starredMessages = response.data || [];
             } catch(e) {
                 console.error("Failed to fetch starred", e);
             }
             this.starredLoading = false;
        },
        async fetchDetails() {
            // Let the caller handle loading and errors
            const response = await this.$axios.get(`/groups/${this.conversationId}`);
            this.groupStub = response.data;
            this.newName = this.groupStub.groupName;
        },

        async saveName() {
            if (!this.newName || this.newName.length < 3) {
                alert("Name must be at least 3 characters.");
                return;
            }
            try {
                await this.$axios.put(`/groups/${this.conversationId}/name`, { name: this.newName });
                this.groupStub.groupName = this.newName;
                this.isEditingName = false;
                this.$emit('update-name', this.newName);
            } catch (e) {
                alert("Error updating name: " + (e.response?.data?.error || e.toString()));
            }
        },

        async triggerPhotoUpload() {
            this.$refs.photoInput.click();
        },

        async onPhotoSelected(e) {
            const file = e.target.files[0];
            if (!file) return;

            this.uploadingPhoto = true;
            let formData = new FormData();
            formData.append("photo", file);

            try {
                await this.$axios.put(`/groups/${this.conversationId}/photo`, formData, {
                    headers: { 'Content-Type': 'multipart/form-data' }
                });
                await this.fetchDetails();
            } catch (e) {
                alert("Error uploading photo: " + e.toString());
            }
            this.uploadingPhoto = false;
        },

        cancelAddMember() {
            this.isAddingMember = false;
            this.newMemberQuery = "";
            this.searchResults = [];
        },

        async searchUser() {
             if (this.newMemberQuery.length < 3) return;
             try {
                let response = await this.$axios.get("/users", { params: { username: this.newMemberQuery } });
                this.searchResults = response.data.filter(u => !this.groupStub.members.includes(u.username));
             } catch(e) {
                 console.error(e);
             }
        },

        async addMember(username) {
            try {
                await this.$axios.post(`/groups/${this.conversationId}/participants`, { username: username }); // Check API call signature
                await this.fetchDetails();
                this.cancelAddMember();
            } catch (e) {
                 alert("Error adding member: " + (e.response?.data?.error || e.toString()));
            }
        },

        async leaveGroup() {
            if (!confirm("Are you sure you want to leave this group?")) return;
            try {
                await this.$axios.delete(`/user/groups/${this.conversationId}`);
                this.$emit('leave');
            } catch (e) {
                alert("Error leaving group: " + e.toString());
            }
        }
    }
}
</script>

<template>
    <div class="modal-backdrop fade show"></div>
    <div class="modal fade show d-block" tabindex="-1">
        <div class="modal-dialog modal-dialog-centered modal-fullscreen-sm-down">
            <div class="modal-content">
                <div class="modal-header">
                    <h5 class="modal-title">{{ realIsGroup ? 'Group Info' : 'Chat Info' }}</h5>
                    <button type="button" class="btn-close" @click="$emit('close')"></button>
                </div>
                <div class="modal-body">
                    <div v-if="loading" class="text-center p-3">
                        <div class="spinner-border text-primary" role="status">
                            <span class="visually-hidden">Loading...</span>
                        </div>
                    </div>
                    
                    <div v-else>
                        <!-- Header with Photo & Name -->
                        <div class="d-flex flex-column align-items-center mb-4">
                            <div class="position-relative mb-2">
                                <img :src="groupStub.groupPhoto || '/feather-sprite-v4.29.0.svg#users'" 
                                     class="rounded-circle border" 
                                     style="width: 150px; height: 150px; object-fit: cover;"
                                     :style="{ padding: groupStub.groupPhoto ? '0' : '30px' }"
                                     alt="Photo">
                                     
                                <button v-if="realIsGroup" class="btn btn-sm btn-light position-absolute bottom-0 end-0 rounded-circle border shadow-sm"
                                        style="width: 32px; height: 32px; padding: 0;"
                                        title="Change Photo"
                                        :disabled="uploadingPhoto"
                                        @click="triggerPhotoUpload">
                                    <svg class="feather w-100 h-100 p-1"><use href="/feather-sprite-v4.29.0.svg#camera"/></svg>
                                </button>
                                <input type="file" ref="photoInput" class="d-none" accept="image/*" @change="onPhotoSelected">
                            </div>

                            <div v-if="!isEditingName" class="d-flex align-items-center gap-2">
                                <h4 class="m-0">{{ groupStub.groupName }}</h4>
                                <button v-if="realIsGroup" class="btn btn-sm btn-link text-secondary p-0" @click="isEditingName = true">
                                    <svg class="feather" style="width: 16px; height: 16px;"><use href="/feather-sprite-v4.29.0.svg#edit-2"/></svg>
                                </button>
                            </div>
                            <div v-else class="d-flex gap-2">
                                <input type="text" v-model="newName" class="form-control form-control-sm" style="max-width: 200px;">
                                <button class="btn btn-sm btn-primary" @click="saveName">Save</button>
                                <button class="btn btn-sm btn-outline-secondary" @click="isEditingName = false">Cancel</button>
                            </div>
                        </div>

                        <!-- Members List (Groups Only) and Starred Messages Tabs -->
                         <ul class="nav nav-tabs mb-3" v-if="realIsGroup">
                             <li class="nav-item">
                                 <a class="nav-link" :class="{active: activeTab === 'members'}" href="#" @click.prevent="activeTab = 'members'">Members</a>
                             </li>
                             <li class="nav-item">
                                 <a class="nav-link" :class="{active: activeTab === 'photos'}" href="#" @click.prevent="activeTab = 'photos'">Photos</a>
                             </li>
                             <li class="nav-item">
                                 <a class="nav-link" :class="{active: activeTab === 'starred'}" href="#" @click.prevent="activeTab = 'starred'">Starred Messages</a>
                             </li>
                         </ul>
                         <ul class="nav nav-tabs mb-3" v-else>
                              <!-- For DMs, we might just show Starred Messages directly or have a single tab -->
                             <li class="nav-item">
                                 <a class="nav-link" :class="{active: activeTab === 'photos'}" href="#" @click.prevent="activeTab = 'photos'">Photos</a>
                             </li>
                             <li class="nav-item">
                                 <a class="nav-link" :class="{active: activeTab === 'starred'}" href="#" @click.prevent="activeTab = 'starred'">Starred Messages</a>
                             </li>
                         </ul>

                        <!-- MEMBERS TAB CONTENT -->
                        <div v-if="realIsGroup && activeTab === 'members'">
                            <div class="d-flex justify-content-between align-items-center mb-2 border-bottom pb-2">
                                 <h6 class="m-0">Members ({{ groupStub.members?.length || 0 }})</h6>
                                 <button v-if="!isAddingMember" class="btn btn-sm btn-link" @click="isAddingMember = true">+ Add</button>
                            </div>
                            
                            <!-- Add Member Section -->
                            <div v-if="isAddingMember" class="mb-3 p-2 bg-light rounded">
                                <div class="input-group input-group-sm mb-2">
                                    <input type="text" class="form-control" v-model="newMemberQuery" placeholder="Search username..." @keyup.enter="searchUser">
                                    <button class="btn btn-outline-secondary" @click="searchUser">Search</button>
                                </div>
                                <div v-if="searchResults.length > 0" class="list-group list-group-flush mb-2" style="max-height: 100px; overflow-y: auto;">
                                    <button v-for="u in searchResults" :key="u.username" class="list-group-item list-group-item-action py-1 d-flex justify-content-between align-items-center" @click="addMember(u.username)">
                                        {{ u.username }}
                                        <svg class="feather text-primary" style="width: 14px; height: 14px;"><use href="/feather-sprite-v4.29.0.svg#plus"/></svg>
                                    </button>
                                </div>
                                <div class="text-end">
                                    <button class="btn btn-sm btn-secondary" @click="cancelAddMember">Close</button>
                                </div>
                            </div>
    
                            <div class="list-group list-group-flush mb-4" style="max-height: 200px; overflow-y: auto;">
                                <div v-for="member in groupStub.members" :key="member" class="list-group-item px-0 d-flex align-items-center gap-2">
                                    <div class="bg-light rounded-circle d-flex align-items-center justify-content-center text-secondary fw-bold" 
                                         style="width: 32px; height: 32px; font-size: 0.8rem;">
                                        {{ member.substring(0, 2).toUpperCase() }}
                                    </div>
                                    <span>{{ member }}</span>
                                    <span v-if="member === $root.username" class="badge bg-secondary ms-auto">You</span>
                                </div>
                            </div>
    
                            <!-- Actions -->
                             <div class="d-grid">
                                <button class="btn btn-outline-danger" @click="leaveGroup">
                                    <svg class="feather me-2"><use href="/feather-sprite-v4.29.0.svg#log-out"/></svg>
                                    Leave Group
                                </button>
                             </div>
                        </div>

                        <!-- PHOTOS TAB CONTENT -->
                        <div v-else-if="activeTab === 'photos'">
                             <div class="d-flex justify-content-between align-items-center mb-2 border-bottom pb-2">
                                 <h6 class="m-0">Shared Photos</h6>
                                 <button class="btn btn-sm btn-link" @click="fetchPhotos">Refresh</button>
                            </div>
                            
                            <div v-if="photosLoading" class="text-center py-3">
                                <div class="spinner-border spinner-border-sm text-secondary"></div>
                            </div>
                            <div v-else-if="photos.length === 0" class="text-center text-muted py-3">
                                <p class="small">No photos shared yet.</p>
                            </div>
                            <div v-else class="container-fluid p-0" style="max-height: 300px; overflow-y: auto;">
                                <div class="row g-2">
                                    <div v-for="(photoUrl, index) in photos" :key="index" class="col-4">
                                        <div class="ratio ratio-1x1 border rounded overflow-hidden position-relative" style="cursor: pointer;" @click="selectedImage = photoUrl">
                                            <img :src="photoUrl" class="w-100 h-100 object-fit-cover">
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>

                        <!-- STARRED TAB CONTENT -->
                        <div v-else class="">
                             <div class="d-flex justify-content-between align-items-center mb-2 border-bottom pb-2">
                                 <h6 class="m-0">Starred Messages</h6>
                                 <button class="btn btn-sm btn-link" @click="fetchStarred">Refresh</button>
                            </div>
                            
                            <div v-if="starredLoading" class="text-center py-3">
                                <div class="spinner-border spinner-border-sm text-secondary"></div>
                            </div>
                            <div v-else-if="starredMessages.length === 0" class="text-center text-muted py-3">
                                <p class="small">No starred messages yet.</p>
                            </div>
                            <div v-else class="list-group list-group-flush" style="max-height: 300px; overflow-y: auto;">
                                <div v-for="msg in starredMessages" :key="msg.messageId" class="list-group-item px-0 py-2">
                                    <div class="d-flex justify-content-between align-items-start">
                                        <div class="small fw-bold">{{ msg.senderName }}</div>
                                        <small class="text-muted" style="font-size: 0.7rem;">{{ new Date(msg.messageTimestamp).toLocaleString() }}</small>
                                    </div>
                                    <div class="text-break small">{{ msg.textContent || (msg.kind === 'image' ? '[Photo]' : '') }}</div>
                                    <div v-if="msg.photoSource" class="mt-1">
                                        <img :src="'/images/' + msg.photoSource" style="height: 60px; border-radius: 4px;">
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>
    
    <ImageViewerModal
        v-if="selectedImage"
        :image-src="selectedImage"
        @close="selectedImage = null"
    />
</template>

<style scoped>
.modal-backdrop {
    background-color: rgba(0, 0, 0, 0.5);
}
</style>
