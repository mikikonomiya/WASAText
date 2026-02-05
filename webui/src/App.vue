<script setup>
import { RouterLink, RouterView } from 'vue-router'
</script>
<script>
import NewChatModal from './components/NewChatModal.vue'
import GroupSettingsModal from './components/GroupSettingsModal.vue'

export default {
    components: {
        NewChatModal,
        GroupSettingsModal
    },
	data() {
		return {
			username: localStorage.getItem("username"),
			conversations: [],
			loading: false,
            showNewChatModal: false,
            
            // Modal state
            showInfoModal: false,
            selectedConvId: null,
            selectedConvName: "",
            selectedConvIsGroup: false,
            selectedConvObj: null,
            selectedConvObj: null,
            myPhoto: null,
            
            // Theme
            darkMode: false,

            // Search
            searchQuery: ""
		}
	},
    computed: {
        filteredConversations() {
            if (!this.searchQuery) {
                return this.conversations;
            }
            const query = this.searchQuery.toLowerCase();
            return this.conversations.filter(c => 
                (c.conversationName || "").toLowerCase().includes(query)
            );
        }
    },
	watch: {
		$route() { // this code runs whenever the route changes
			this.username = localStorage.getItem("username");
			if (this.username) {
				this.refreshConversations();
                this.fetchMyProfile();
			} else {
                this.conversations = [];
                this.myPhoto = null;
            }
		}
	},
    methods: {
        closeSidebar() {
            // Check if mobile (sidebar is collapsible)
            if (window.innerWidth < 768) { 
                const sidebar = document.getElementById('sidebarMenu');
                if (sidebar && sidebar.classList.contains('show')) {
                    // Bootstrap 5 native toggle
                    sidebar.classList.remove('show');
                }
            }
        },
        toggleTheme() {
            this.darkMode = !this.darkMode;
            const theme = this.darkMode ? 'dark' : 'light';
            document.documentElement.setAttribute('data-bs-theme', theme);
            localStorage.setItem('theme', theme);
        },
        initTheme() {
            const stored = localStorage.getItem('theme');
            if (stored) {
                this.darkMode = stored === 'dark';
            } else {
                this.darkMode = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
            }
            document.documentElement.setAttribute('data-bs-theme', this.darkMode ? 'dark' : 'light');
        },

        openNewChatModal() {
            this.showNewChatModal = true;
        },
        openInfoModal(c) {
            this.selectedConvId = c.conversationId;
            this.selectedConvName = c.conversationName; // Might be null for DMs in DB, but usually filled by backend? 
            // Wait, backend 'GetMyConversations' FILLS c.Name (Step 1718). YES.
            // So c.conversationName IS populated even for DMs.
            // We need to distinguish Group vs DM roughly.
            // The backend fills c.Name for DMs using `otherUsername`.
            // But how do we know if it is a Group?
            // "Groups" usually have a non-empty name in the DB. "DMs" have empty name.
            // But the JSON response always has `conversationName`.
            // We can check if `c.conversationPhoto` is present? No, DMs can have photos now.
            // Maybe we assume: If I can "Leave" it, it's a group.
            // But we don't know that here.
            // Let's rely on the heuristic: "Does it look like a group?"
            // OR we can pass a prop `isGroup` based on... nothing solid in `conversations` list response.
            // Wait, `GetMyConversations` returns `Conversation`.
            // If we look at `c.Name` in `GetMyConversations`, for DMs it is set to `otherUsername`.
            // Ideally we should have a `type` field.
            // For now, let's treat *everything* as potentially having info.
            // `GroupSettingsModal` handles `isGroup` prop to hide "Add Member" / "Leave".
            // How do we set `isGroup` properly?
            // If the user created a Group named "Alice", it looks like a DM.
            // Let's assume everything is a group unless we know otherwise? No, that enables "Add Member".
            // Safe bet: Pass `false` for `isGroup` unless we are sure?
            // Now we can rely on the backend `isGroup` field!
            this.selectedConvIsGroup = !!c.isGroup; 
            this.selectedConvObj = c;
            this.showInfoModal = true;
        },
        handleChatCreated(conversationId) {
            this.refreshConversations();
            this.$router.push('/conversations/' + conversationId);
        },
		async refreshConversations() {
			if (!this.username) return;
			this.loading = true;
			try {
				let response = await this.$axios.get("/conversations");
				this.conversations = response.data;
			} catch (e) {
				console.error(e);
			}
			this.loading = false;
		},
        async fetchMyProfile() {
            try {
                if (!this.username) return;
                let response = await this.$axios.get(`/users/${this.username}`);
                this.myPhoto = response.data.photo_id ? `/images/${response.data.photo_id}` : null;
            } catch (e) {
                console.error("Failed to fetch my profile", e);
            }
        },
		logout() {
			localStorage.removeItem("token");
			localStorage.removeItem("username");
			this.$router.push("/login");
		},
		formatTime(timestamp) {
			if (!timestamp) return "";
			return new Date(timestamp).toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'});
		}
	},
	mounted() {
        this.initTheme();
		if (this.username) {
			this.refreshConversations();
            this.fetchMyProfile();
		}
		// Poll for new conversations every 5 seconds
		setInterval(() => {
			if (this.username) {
				this.refreshConversations();
			}
		}, 5000);
	}
}
</script>

<template>
	<div v-if="$route.path === '/login'" class="container-fluid p-0">
		<RouterView />
	</div>
	
	<div v-else class="container-fluid d-flex flex-column h-100 p-0 overflow-hidden">
        <NewChatModal :show="showNewChatModal" @close="showNewChatModal = false" @chat-created="handleChatCreated" />
        
        <!-- Mobile Header -->
        <header class="navbar navbar-dark sticky-top bg-dark flex-md-nowrap p-0 shadow d-md-none z-index-1030 flex-shrink-0">
            <a class="navbar-brand col-12 me-0 px-3 fs-6" href="#">WASAText</a>
            <button class="navbar-toggler position-absolute d-md-none collapsed" type="button" data-bs-toggle="collapse" data-bs-target="#sidebarMenu" aria-controls="sidebarMenu" aria-expanded="false" aria-label="Toggle navigation" style="top: 0.5rem; right: 1rem;">
                <span class="navbar-toggler-icon"></span>
            </button>
        </header>

        <GroupSettingsModal 
            v-if="showInfoModal"
            :conversation-id="selectedConvId"
            :initial-name="selectedConvName"
            :conversation="selectedConvObj"
            :is-group="selectedConvIsGroup"
            @close="showInfoModal = false"
            @update-name="refreshConversations"
            @leave="() => { showInfoModal = false; refreshConversations(); }"
        />

		<div class="row flex-grow-1 g-0 overflow-hidden">
			<nav id="sidebarMenu" class="col-md-3 col-lg-2 d-md-block bg-body-tertiary sidebar collapse h-100">
				<div class="position-sticky pt-3 sidebar-sticky d-flex flex-column h-100">
					<h6 class="sidebar-heading d-flex justify-content-between align-items-center px-3 mt-4 mb-1 text-muted text-uppercase">
						<span>WASAText</span>
                        <div class="d-flex gap-1">
                            <button class="btn btn-sm btn-outline-secondary" @click="toggleTheme" :title="darkMode ? 'Light Mode' : 'Dark Mode'">
                                <svg v-if="darkMode" class="feather"><use href="/feather-sprite-v4.29.0.svg#sun"/></svg>
                                <svg v-else class="feather"><use href="/feather-sprite-v4.29.0.svg#moon"/></svg>
                            </button>
                            <button class="btn btn-sm btn-outline-secondary" @click="refreshConversations" title="Refresh">
                                <svg class="feather"><use href="/feather-sprite-v4.29.0.svg#refresh-cw"/></svg>
                            </button>
                        </div>
					</h6>
					
                    <!-- Profile Link -->
                    <div class="mx-3 mt-2 mb-3 p-2 rounded user-profile-card d-flex align-items-center justify-content-between">
                        <div class="d-flex align-items-center gap-2 text-truncate">
                            <div class="bg-primary text-white rounded-circle d-flex align-items-center justify-content-center overflow-hidden" style="width: 32px; height: 32px; font-size: 14px;">
                                <img v-if="myPhoto" :src="myPhoto" alt="" style="width: 100%; height: 100%; object-fit: cover;">
                                <span v-else class="fw-bold">{{ username?.charAt(0).toUpperCase() }}</span>
                            </div>
                            <span class="fw-bold text-body text-truncate">{{ username }}</span>
                        </div>
                        <RouterLink to="/profile" class="btn btn-sm btn-link text-secondary p-0" title="My Profile">
                             <svg class="feather" style="width: 16px; height: 16px;"><use href="/feather-sprite-v4.29.0.svg#settings"/></svg>
                        </RouterLink>
                    </div>

                    <!-- Search Bar -->
                    <div class="px-3 mb-2">
                        <div class="input-group input-group-sm">
                            <span class="input-group-text bg-body border-end-0">
                                <svg class="feather" style="width: 14px; height: 14px;"><use href="/feather-sprite-v4.29.0.svg#search"/></svg>
                            </span>
                            <input type="text" class="form-control border-start-0 ps-0" placeholder="Search chats..." v-model="searchQuery">
                        </div>
                    </div>
					
					<ul class="nav flex-column mb-auto overflow-auto">
						<li class="nav-item" v-for="c in filteredConversations" :key="c.conversationId">
							<div class="nav-link d-flex align-items-center gap-2 p-0" :class="{ 'bg-primary-subtle': $route.path === '/conversations/' + c.conversationId }">
                                <!-- Avatar Clickable Area -->
                                <div class="p-2 cursor-pointer" @click.stop.prevent="openInfoModal(c)" title="View Photo">
                                    <div style="width: 32px; height: 32px; border-radius: 50%; background: #ccc; overflow: hidden; flex-shrink: 0;" class="border">
                                        <img v-if="c.conversationPhoto" :src="c.conversationPhoto" alt="" style="width: 100%; height: 100%; object-fit: cover;">
                                        <span v-else class="d-flex justify-content-center align-items-center h-100 small text-white">{{ (c.conversationName || '?').charAt(0) }}</span>
                                    </div>
                                </div>
                                
                                
                                <RouterLink :to="'/conversations/' + c.conversationId" class="text-decoration-none flex-grow-1 text-body py-2 pe-2 overflow-hidden" @click="closeSidebar">
    								<div class="text-truncate">
    									<div class="d-flex justify-content-between">
    										<span class="fw-bold text-truncate">{{ c.conversationName || 'Unknown Conversation' }}</span>
    										<small class="text-body-secondary" style="font-size: 0.75rem;">{{ formatTime(c.lastMessageTimestamp) }}</small>
    									</div>
    									<div class="d-flex justify-content-between align-items-center">
    										<div class="small text-body-secondary text-truncate" v-if="c.lastMessagePreview" style="max-width: 80%;">{{ c.lastMessagePreview }}</div>
    										<span v-if="c.unreadCount > 0" class="badge bg-danger rounded-pill">{{ c.unreadCount }}</span>
    									</div>
    								</div>
                                </RouterLink>
							</div>
						</li>
						<li class="nav-item mt-3 px-3">
							<a href="#" class="btn btn-primary w-100 btn-sm text-white d-flex align-items-center justify-content-center gap-2" @click.prevent="openNewChatModal">
								<svg class="feather"><use href="/feather-sprite-v4.29.0.svg#plus-circle"/></svg>
								New Chat
							</a>
						</li>
					</ul>
					
					<div class="mt-auto p-3 border-top">
						<button @click="logout" class="btn btn-outline-danger w-100 btn-sm">
							<svg class="feather"><use href="/feather-sprite-v4.29.0.svg#log-out"/></svg>
							Log out
						</button>
					</div>
				</div>
			</nav>

			<main class="col-md-9 ms-sm-auto col-lg-10 h-100 p-0">
				<RouterView @profile-updated="fetchMyProfile" />
			</main>
		</div>
	</div>
</template>

<style>
/* Ensure the sidebar and main content take full height */
#app {
	height: 100vh;
	overflow: hidden;
}
.sidebar {
	height: 100vh;
	/* overflow-y: auto; handled by flex column now */ 
	box-shadow: inset -1px 0 0 rgba(0, 0, 0, .1);
}
main {
	height: 100%;
	overflow: hidden; /* Let child views manage scrolling */
}
.cursor-pointer {
    cursor: pointer;
}
.z-index-1030 {
    z-index: 1030;
}
/* Style adjustments for search input */
.input-group-text {
    border-right: none;
    background-color: var(--bs-body-bg);
}
.form-control:focus {
    box-shadow: none;
    border-color: var(--bs-border-color);
}

.user-profile-card {
    background-color: var(--bs-body-bg);
    border: 1px solid var(--bs-border-color);
    box-shadow: 0 0.25rem 0.75rem rgba(0, 0, 0, 0.08) !important;
}
.hover-opacity-100:hover {
    opacity: 1 !important;
}
</style>
