<script>
import ImageViewerModal from '../components/ImageViewerModal.vue'

export default {
    components: { ImageViewerModal },
    data() {
        return {
            loading: false,
            username: localStorage.getItem("username"),
            identifier: "",
            photoUrl: null,
            
            isEditingName: false,
            newName: "",
            
            isUploadingPhoto: false,
            selectedFile: null,
            uploadPreview: null,
            
            error: null,
            success: null,

            // Global Starred
            starredMessages: [],
            starredLoading: false,
            selectedImage: null
        }
    },
    methods: {
        async fetchStarredMessages() {
            this.starredLoading = true;
            try {
                const response = await this.$axios.get("/starred");
                this.starredMessages = response.data || [];
            } catch (e) {
                console.error("Failed to fetch starred messages", e);
            }
            this.starredLoading = false;
        },

        async fetchProfile() {
            this.loading = true;
            try {
                // Get my own profile
                const response = await this.$axios.get(`/users/${this.username}`);
                this.identifier = response.data.identifier;
                // photo_id from API -> construct URL
                // Backend sends `photo_id` field in User struct (from search/get user). Wait, verifying User struct.
                // In database.go: User struct has { identifier, username }. It DOES NOT have photo_id in the JSON response?
                // Let's check `get-user-profile.go` and `database.go`. 
                // Step 1643 view of database.go lines 228-232 shows User struct:
                // type User struct { Identifier string, Username string }
                // It seems the User struct returned by Search/Get might NOT have the photo!?
                // BUT `users` table has `photo_id`.
                // I might need to update the User struct in Go if I want to display the photo here. 
                // OR I can use the conversation photo logic if I had a chat with myself? No.
                
                // Let's assume for now I might need to fix the backend to return photo_id. 
                // I'll write the frontend to expect it, and if it's missing, I'll fix the backend in the next step.
                if (response.data.photo_id) {
                     this.photoUrl = "/images/" + response.data.photo_id;
                } else {
                    this.photoUrl = null;
                }
            } catch (e) {
                this.error = "Failed to load profile.";
                console.error(e);
            }
            this.loading = false;
        },
        
        startEditName() {
            this.newName = this.username;
            this.isEditingName = true;
            this.error = null;
            this.success = null;
        },
        
        cancelEditName() {
            this.isEditingName = false;
            this.newName = "";
        },
        
        async saveName() {
            if (!this.newName || this.newName.length < 3) {
                this.error = "Username must be at least 3 characters.";
                return;
            }
            
            try {
                await this.$axios.put("/user/name", { name: this.newName });
                
                // Success
                this.username = this.newName;
                localStorage.setItem("username", this.newName);
                this.success = "Username updated!";
                this.isEditingName = false;
                
                // Refresh full app state if possible (router reload?)
                // Ideally propagate to App.vue via event or just reload page
                this.$emit('profile-updated');
                
            } catch (e) {
                if (e.response && e.response.status === 409) {
                    this.error = "Username already taken.";
                } else {
                    this.error = "Error updating name: " + (e.response?.statusText || e.toString());
                }
            }
        },
        
        triggerFileInput() {
            this.$refs.fileInput.click();
        },
        
        onFileSelected(e) {
            const file = e.target.files[0];
            if (!file) return;
            
            this.selectedFile = file;
            this.uploadPreview = URL.createObjectURL(file);
            this.isUploadingPhoto = true;
            this.error = null;
            this.success = null;
        },
        
        async savePhoto() {
            if (!this.selectedFile) return;
            
            let formData = new FormData();
            formData.append("photo", this.selectedFile);
            
            try {
                await this.$axios.put("/user/photo", formData, {
                    headers: { 'Content-Type': 'multipart/form-data' }
                });
                
                this.success = "Photo updated!";
                this.isUploadingPhoto = false;
                this.selectedFile = null;
                this.uploadPreview = null;
                await this.fetchProfile(); // Refresh to show new photo
                
                this.$emit('profile-updated');
            } catch (e) {
                this.error = "Error uploading photo: " + (e.response?.data || e.toString());
            }
        },
        
        cancelPhoto() {
            this.selectedFile = null;
            this.uploadPreview = null;
            this.isUploadingPhoto = false;
        },
        formatDate(ts) {
            if (!ts) return "";
            // Handle ISO format (already has T and Z)
            if (ts.includes("T") && ts.includes("Z")) {
                try {
                    return new Date(ts).toLocaleString();
                } catch (e) {
                    return ts;
                }
            }
            
            // Handle SQL format (needs T and Z)
            const iso = ts.replace(" ", "T") + (ts.includes("Z") ? "" : "Z"); 
            try {
                let d = new Date(iso);
                if (isNaN(d.getTime())) return ts; // Return original if parsing fails
                return d.toLocaleString();
            } catch (e) {
                return ts;
            }
        }
    },
    mounted() {
        this.fetchProfile();
        this.fetchStarredMessages();
    }
}
</script>

<template>
    <div class="container py-5 h-100 overflow-auto">
        <div class="row justify-content-center">
            <div class="col-md-8 col-lg-6">
                <div class="card shadow-sm">
                    <div class="card-header bg-body-tertiary">
                        <h4 class="mb-0">My Profile</h4>
                    </div>
                    <div class="card-body text-center">
                        
                        <!-- Alerts -->
                        <div v-if="error" class="alert alert-danger">{{ error }}</div>
                        <div v-if="success" class="alert alert-success">{{ success }}</div>
                        
                        <!-- Photo Section -->
                        <div class="mb-4 position-relative d-inline-block">
                            <div class="profile-photo-container rounded-circle overflow-hidden mb-3 mx-auto border" style="width: 150px; height: 150px;">
                                <img v-if="isUploadingPhoto && uploadPreview" :src="uploadPreview" class="w-100 h-100 object-fit-cover">
                                <img v-else-if="photoUrl" :src="photoUrl" class="w-100 h-100 object-fit-cover">
                                <div v-else class="w-100 h-100 bg-secondary d-flex align-items-center justify-content-center text-white display-4">
                                    {{ username?.charAt(0).toUpperCase() }}
                                </div>
                            </div>
                            
                            <div v-if="!isUploadingPhoto">
                                <button class="btn btn-sm btn-outline-primary" @click="triggerFileInput">
                                    <svg class="feather"><use href="/feather-sprite-v4.29.0.svg#camera"/></svg> Change Photo
                                </button>
                                <input type="file" ref="fileInput" class="d-none" accept="image/*" @change="onFileSelected">
                            </div>
                            <div v-else class="d-flex justify-content-center gap-2">
                                <button class="btn btn-sm btn-primary" @click="savePhoto">Save</button>
                                <button class="btn btn-sm btn-outline-secondary" @click="cancelPhoto">Cancel</button>
                            </div>
                        </div>
                        
                        <!-- Info Section -->
                        <div class="mb-3">
                            <label class="form-label text-muted small text-uppercase">Username</label>
                            
                            <div v-if="!isEditingName" class="d-flex align-items-center justify-content-center gap-2">
                                <h3 class="m-0">{{ username }}</h3>
                                <button class="btn btn-sm btn-link text-muted" @click="startEditName">
                                    <svg class="feather"><use href="/feather-sprite-v4.29.0.svg#edit-2"/></svg>
                                </button>
                            </div>
                            
                            <div v-else class="d-flex justify-content-center gap-2">
                                <input type="text" v-model="newName" class="form-control form-control-lg text-center" style="max-width: 250px;">
                                <button class="btn btn-success" @click="saveName">
                                    <svg class="feather"><use href="/feather-sprite-v4.29.0.svg#check"/></svg>
                                </button>
                                <button class="btn btn-outline-secondary" @click="cancelEditName">
                                    <svg class="feather"><use href="/feather-sprite-v4.29.0.svg#x"/></svg>
                                </button>
                            </div>
                        </div>
                        
                        <div class="text-muted small">
                            ID: {{ identifier }}
                        </div>
                        
                    </div>
                </div>
            </div>
        </div>
        <div class="row justify-content-center mt-4">
            <div class="col-md-8 col-lg-6">
                <div class="card shadow-sm">
                    <div class="card-header bg-body-tertiary d-flex justify-content-between align-items-center">
                        <h5 class="mb-0">Starred Messages</h5>
                        <button class="btn btn-sm btn-link" @click="fetchStarredMessages">Refresh</button>
                    </div>
                    <div class="card-body p-0">
                         <div v-if="starredLoading" class="text-center py-4">
                            <div class="spinner-border spinner-border-sm text-secondary"></div>
                        </div>
                        <div v-else-if="starredMessages.length === 0" class="text-center text-muted py-4">
                            <p class="small mb-0">You haven't starred any messages yet.</p>
                        </div>
                        <div v-else style="max-height: 400px; overflow-y: auto;">
                            <div class="list-group list-group-flush"> 
                                <div v-for="msg in starredMessages" :key="msg.messageId" class="list-group-item">
                                    <div class="d-flex justify-content-between align-items-start mb-1">
                                        <span class="fw-bold small">{{ msg.senderName || msg.senderId || 'Unknown User' }}</span>
                                        <small class="text-muted" style="font-size: 0.7rem;">{{ formatDate(msg.messageTimestamp) }}</small>
                                    </div>
                                    <div class="text-break small">{{ msg.textContent || (msg.kind === 'image' ? '[Photo]' : 'Message') }}</div>
                                    <div v-if="msg.photoSource" class="mt-2">
                                        <img :src="'/images/' + msg.photoSource" style="height: 100px; border-radius: 4px; object-fit: cover; cursor: pointer;"
                                             @click="selectedImage = '/images/' + msg.photoSource">
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
.object-fit-cover {
    object-fit: cover;
}
</style>
