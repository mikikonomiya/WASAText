<script>
export default {
    props: ['imageSrc', 'fileName'],
    emits: ['close'],
    methods: {
        close() {
            this.$emit('close');
        },
        async downloadImage() {
            try {
                const response = await fetch(this.imageSrc);
                const blob = await response.blob();
                const url = window.URL.createObjectURL(blob);
                const a = document.createElement('a');
                a.style.display = 'none';
                a.href = url;
                // Use provided fileName or fallback
                a.download = this.fileName || 'downloaded-image.jpg';
                document.body.appendChild(a);
                a.click();
                window.URL.revokeObjectURL(url);
                document.body.removeChild(a);
            } catch (e) {
                console.error("Download failed", e);
                alert("Failed to download image");
            }
        }
    }
}
</script>

<template>
    <div class="image-viewer-overlay" @click.self="close">
        <div class="position-absolute top-0 end-0 p-3 d-flex gap-2">
            <button class="btn btn-dark rounded-circle" @click="downloadImage" title="Download">
                <svg class="feather text-white"><use href="/feather-sprite-v4.29.0.svg#download"/></svg>
            </button>
            <button class="btn btn-dark rounded-circle" @click="close" title="Close">
                <svg class="feather text-white"><use href="/feather-sprite-v4.29.0.svg#x"/></svg>
            </button>
        </div>
        
        <div class="image-container">
            <img :src="imageSrc" class="img-fluid" style="max-height: 90vh; max-width: 90vw; object-fit: contain;">
        </div>
    </div>
</template>

<style scoped>
.image-viewer-overlay {
    position: fixed;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    background-color: rgba(0, 0, 0, 0.9);
    z-index: 9999;
    display: flex;
    align-items: center;
    justify-content: center;
    animation: fadeIn 0.2s ease-out;
}

.image-container img {
    box-shadow: 0 0 20px rgba(0,0,0,0.5);
}

@keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
}
</style>
