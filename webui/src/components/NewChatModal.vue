<script>
export default {
	props: ['show'],
	emits: ['close', 'chat-created'],
	data() {
		return {
			mode: 'dm', // 'dm' or 'group'
			userSearchQuery: "",
			searchResults: [],
			selectedUser: null,
			
			// Group data
			groupName: "",
			selectedGroupMembers: [],
			
			errorMsg: null,
            loading: false,
		}
	},
	methods: {
		close() {
			this.$emit('close');
			this.reset();
		},
		reset() {
			this.mode = 'dm';
			this.userSearchQuery = "";
			this.searchResults = [];
			this.selectedUser = null;
			this.groupName = "";
			this.selectedGroupMembers = [];
			this.errorMsg = null;
            this.loading = false;
		},
		async searchUsers() {
			if (this.userSearchQuery.length < 3) return;
            this.loading = true;
            this.errorMsg = null;
			try {
				let response = await this.$axios.get("/users", { params: { username: this.userSearchQuery } });
				// Filter out myself
				const myName = localStorage.getItem("username");
				this.searchResults = response.data.filter(u => u.username !== myName);
			} catch (e) {
				this.errorMsg = "Error searching users";
			}
            this.loading = false;
		},
		selectUserForDM(user) {
			this.selectedUser = user;
			this.createDM();
		},
		async createDM() {
			if (!this.selectedUser) return;
			try {
				let response = await this.$axios.post("/conversations", { targetUsername: this.selectedUser.username });
				this.$emit('chat-created', response.data.conversationId);
				this.close();
			} catch (e) {
				this.errorMsg = e.response?.data?.message || e.toString();
			}
		},
		toggleGroupMember(user) {
			const idx = this.selectedGroupMembers.findIndex(u => u.username === user.username);
			if (idx === -1) {
				this.selectedGroupMembers.push(user);
			} else {
				this.selectedGroupMembers.splice(idx, 1);
			}
		},
		async createGroup() {
			if (!this.groupName || this.selectedGroupMembers.length === 0) {
                this.errorMsg = "Please enter a group name and select at least one member.";
                return;
            }
			try {
                const memberUsernames = this.selectedGroupMembers.map(u => u.username);
				let response = await this.$axios.post("/groups", { 
                    groupName: this.groupName,
                    memberUsernames: memberUsernames
                });
                // Backend returns a Conversation object, just like DM.
				this.$emit('chat-created', response.data.conversationId); 
				this.close();
			} catch (e) {
				this.errorMsg = e.response?.data?.message || e.toString();
			}
		}
	}
}
</script>

<template>
	<div v-if="show" class="modal-backdrop d-block" style="background-color: rgba(0,0,0,0.5);"></div>
	<div v-if="show" class="modal d-block" tabindex="-1">
		<div class="modal-dialog modal-dialog-centered modal-fullscreen-sm-down">
			<div class="modal-content">
				<div class="modal-header">
					<h5 class="modal-title">New Conversation</h5>
					<button type="button" class="btn-close" @click="close"></button>
				</div>
				<div class="modal-body">
					<!-- Tabs -->
					<ul class="nav nav-tabs mb-3">
						<li class="nav-item">
							<a class="nav-link" :class="{active: mode === 'dm'}" href="#" @click.prevent="mode = 'dm'">Direct Message</a>
						</li>
						<li class="nav-item">
							<a class="nav-link" :class="{active: mode === 'group'}" href="#" @click.prevent="mode = 'group'">New Group</a>
						</li>
					</ul>

					<div v-if="mode === 'dm'">
						<div class="mb-3">
							<label class="form-label">Search User</label>
							<div class="input-group">
                                <input type="text" class="form-control" v-model="userSearchQuery" @keyup.enter="searchUsers" placeholder="Type username...">
                                <button class="btn btn-outline-secondary" @click="searchUsers">Search</button>
                            </div>
						</div>
                        <div v-if="loading" class="text-center"><LoadingSpinner/></div>
						<div class="list-group">
							<button v-for="user in searchResults" :key="user.username" 
                                class="list-group-item list-group-item-action d-flex align-items-center"
                                @click="selectUserForDM(user)">
								{{ user.username }}
							</button>
                            <div v-if="searchResults.length === 0 && userSearchQuery.length > 2 && !loading" class="text-muted small">
                                No users found.
                            </div>
						</div>
					</div>

					<div v-if="mode === 'group'">
						<div class="mb-3">
							<label class="form-label">Group Name</label>
							<input type="text" class="form-control" v-model="groupName" placeholder="My Cool Group">
						</div>
						<div class="mb-3">
							<label class="form-label">Add Members</label>
                            <div class="input-group mb-2">
                                <input type="text" class="form-control" v-model="userSearchQuery" @keyup.enter="searchUsers" placeholder="Search users...">
                                <button class="btn btn-outline-secondary" @click="searchUsers">Search</button>
                            </div>
							<div class="list-group" style="max-height: 200px; overflow-y: auto;">
								<button v-for="user in searchResults" :key="user.username" 
                                    class="list-group-item list-group-item-action d-flex justify-content-between align-items-center"
                                    @click="toggleGroupMember(user)">
									{{ user.username }}
                                    <svg v-if="selectedGroupMembers.some(u => u.username === user.username)" class="feather text-primary"><use href="/feather-sprite-v4.29.0.svg#check"/></svg>
								</button>
							</div>
                            <div class="mt-2" v-if="selectedGroupMembers.length > 0">
                                <small>Selected: {{ selectedGroupMembers.map(u => u.username).join(', ') }}</small>
                            </div>
						</div>
                        <button class="btn btn-primary w-100" @click="createGroup">Create Group</button>
					</div>

					<ErrorMsg v-if="errorMsg" :msg="errorMsg" class="mt-3"></ErrorMsg>
				</div>
			</div>
		</div>
	</div>
</template>
