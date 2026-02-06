<script>
export default {
    props: {
        reactions: {
            type: Array,
            default: () => []
        }
    },
    emits: ['close']
}
</script>

<template>
    <div class="modal-backdrop fade show"></div>
    <div class="modal fade show d-block" tabindex="-1">
        <div class="modal-dialog modal-sm modal-dialog-centered">
            <div class="modal-content">
                <div class="modal-header border-0 pb-2">
                    <h5 class="modal-title fs-6">Reactions</h5>
                    <button type="button" class="btn-close" @click="$emit('close')"></button>
                </div>
                <div class="modal-body pt-0" style="max-height: 300px; overflow-y: auto;">
                    <div v-if="reactions.length === 0" class="text-center text-muted small py-3">
                        No reactions yet.
                    </div>
                    
                    <div v-else class="d-flex flex-column gap-2">
                        <div v-for="(r, i) in reactions" :key="i" class="d-flex align-items-center justify-content-between p-2 rounded hover-bg">
                            <span class="fw-bold text-dark">{{ r.username }}</span>
                            <span class="fs-5">{{ r.emoji }}</span>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>

<style scoped>
.modal-backdrop {
    background-color: rgba(0, 0, 0, 0.5);
    z-index: 1055;
}
.modal {
    z-index: 1060;
}
.hover-bg:hover {
    background-color: #f8f9fa;
}
</style>
