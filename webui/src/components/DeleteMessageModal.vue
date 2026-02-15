<script>
export default {
    props: ['message', 'isMe'], // input from parent
    emits: ['close', 'confirm'], // output to parent
    methods: {
        confirm(scope) {
            this.$emit('confirm', this.message.messageId, scope);
        }
    }
}
</script>

<template>
    <div class="modal-backdrop fade show"></div>
    <div class="modal fade show d-block" tabindex="-1">
        <div class="modal-dialog modal-sm modal-dialog-centered">
            <div class="modal-content">
                <div class="modal-header border-0 pb-0">
                    <h5 class="modal-title fs-6">Delete Message?</h5>
                    <button type="button" class="btn-close" @click="$emit('close')"></button>
                </div>
                <!-- Debug info (hidden in prod, but useful now if needed) -->
                <!-- <div style="display:none">IsMe: {{ isMe }}</div> -->
                <div class="modal-body">
                    <p class="small text-muted mb-3">Which deletion method do you want to use?</p>
                    
                    <div class="d-grid gap-2">
                        <button class="btn btn-outline-danger btn-sm" @click="confirm('me')">
                            Delete for me
                        </button>
                        <button v-if="isMe" class="btn btn-danger btn-sm" @click="confirm('everyone')">
                            Delete for everyone
                        </button>
                    </div>
                </div>
                <div class="modal-footer border-0 pt-0">
                    <button type="button" class="btn btn-sm btn-secondary w-100" @click="$emit('close')">Cancel</button>
                </div>
            </div>
        </div>
    </div>
</template>

<style scoped>
.modal-backdrop {
    background-color: rgba(0, 0, 0, 0.5);
}
</style>
