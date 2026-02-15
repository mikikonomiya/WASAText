<script>
export default {
    props: {
        conversations: Array,
        currentConversationId: String
    },
    data() {
        return {
            searchQuery: "",
            selectedConvId: null
        }
    },
    computed: {
        filteredConversations() {
            let list = this.conversations;
            if (this.currentConversationId) {
                list = list.filter(c => c.conversationId !== this.currentConversationId);
            }
            if (!this.searchQuery) return list;
            const q = this.searchQuery.toLowerCase();
            return list.filter(c => 
                (c.conversationName && c.conversationName.toLowerCase().includes(q))
            );
        }
    },
    methods: {
        select(convId) {
            this.selectedConvId = convId;
        },
        confirm() {
            if (this.selectedConvId) {
                this.$emit('confirm', this.selectedConvId); // emit the selected conversation ID to the parent component
            }
        }
    }
}
</script>

<template>
    <div class="modal fade show d-block" style="background: rgba(0,0,0,0.5);" tabindex="-1">
        <div class="modal-dialog modal-dialog-centered">
            <div class="modal-content">
                <div class="modal-header">
                    <h5 class="modal-title">Forward Message</h5>
                    <button type="button" class="btn-close" @click="$emit('close')"></button>
                </div>
                <div class="modal-body p-0">
                    <div class="p-2 border-bottom">
                         <input type="text" class="form-control form-control-sm" placeholder="Search chats..." v-model="searchQuery">
                    </div>
                    <div class="list-group list-group-flush" style="max-height: 300px; overflow-y: auto;">
                        <button v-for="c in filteredConversations" :key="c.conversationId" 
                            class="list-group-item list-group-item-action d-flex align-items-center justify-content-between"
                            :class="{'active': selectedConvId === c.conversationId}"
                            @click="select(c.conversationId)">
                            
                            <span class="text-truncate">{{ c.conversationName || 'Chat ' + c.conversationId }}</span>
                            <svg v-if="selectedConvId === c.conversationId" class="feather text-white"><use href="/feather-sprite-v4.29.0.svg#check"/></svg>
                        </button>
                        <div v-if="filteredConversations.length === 0" class="p-3 text-center text-muted small">
                            No chats found.
                        </div>
                    </div>
                </div>
                <div class="modal-footer">
                    <button type="button" class="btn btn-secondary btn-sm" @click="$emit('close')">Cancel</button>
                    <button type="button" class="btn btn-primary btn-sm" @click="confirm" :disabled="!selectedConvId">
                        Forward
                    </button>
                </div>
            </div>
        </div>
    </div>
</template>
