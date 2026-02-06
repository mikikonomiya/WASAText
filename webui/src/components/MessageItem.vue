<script>
export default {
    props: {
        message: Object,
        isMe: Boolean,
        selectable: Boolean,
        selected: Boolean
    },
    emits: ['reply', 'forward', 'toggle-selection', 'react', 'toggle-star', 'view-image', 'delete', 'view-reactions'],
    methods: {
        formatTime(timestamp) {
            return new Date(timestamp).toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'});
        }
    }
}
</script>

<template>
    <!-- Message Container: Flex row to position bubbles + Action Button -->
    <div class="d-flex w-100 position-relative group-message-container" :class="{'justify-content-end': isMe, 'justify-content-start': !isMe}">
        
        <!-- Action Buttons (Hidden by default, shown on hover) -->
        <div class="action-buttons opacity-0 align-self-center mx-2 d-flex gap-1" :class="{'order-last': !isMe}">
            <button class="btn btn-sm btn-light rounded-circle shadow-sm" @click="$emit('toggle-star', message)" :title="message.isStarred ? 'Unstar' : 'Star'">
                 <svg class="feather" style="width: 14px; height: 14px;" :class="{'text-warning': message.isStarred}" :style="{ fill: message.isStarred ? 'currentColor' : 'none' }">
                    <use href="/feather-sprite-v4.29.0.svg#star"/>
                 </svg>
            </button>
            <button class="btn btn-sm btn-light rounded-circle shadow-sm" @click="$emit('reply', message)" title="Reply">
                <svg class="feather" style="width: 14px; height: 14px;"><use href="/feather-sprite-v4.29.0.svg#corner-up-left"/></svg>
            </button>
            <button class="btn btn-sm btn-light rounded-circle shadow-sm" @click="$emit('forward', message)" title="Forward">
                <svg class="feather" style="width: 14px; height: 14px;"><use href="/feather-sprite-v4.29.0.svg#arrow-right"/></svg>
            </button>
            <div class="dropdown">
                <button class="btn btn-sm btn-light rounded-circle shadow-sm" data-bs-toggle="dropdown" aria-expanded="false" title="React">
                     <svg class="feather" style="width: 14px; height: 14px;"><use href="/feather-sprite-v4.29.0.svg#smile"/></svg>
                </button>
                <ul class="dropdown-menu p-2 shadow" style="min-width: 180px;">
                    <li class="d-flex justify-content-between">
                        <button class="btn btn-sm btn-light fs-5" @click="$emit('react', message, '👍')">👍</button>
                        <button class="btn btn-sm btn-light fs-5" @click="$emit('react', message, '❤️')">❤️</button>
                        <button class="btn btn-sm btn-light fs-5" @click="$emit('react', message, '😂')">😂</button>
                        <button class="btn btn-sm btn-light fs-5" @click="$emit('react', message, '😮')">😮</button>
                        <button class="btn btn-sm btn-light fs-5" @click="$emit('react', message, '😢')">😢</button>
                    </li>
                </ul>
            </div>
            <button class="btn btn-sm btn-light rounded-circle shadow-sm text-danger" @click="$emit('delete', message)" title="Delete">
                <svg class="feather" style="width: 14px; height: 14px;"><use href="/feather-sprite-v4.29.0.svg#trash-2"/></svg>
            </button>
        </div>

        <div class="card shadow-sm mb-1 position-relative" style="max-width: 75%; min-width: 80px;" 
             :class="{'bg-myself border-0': isMe, 'bg-message-received': !isMe, 'border border-primary border-3': selected}"
             @click="selectable ? $emit('toggle-selection', message) : null"
             :style="selectable ? 'cursor: pointer;' : ''">
             
            <div class="card-body p-2">
                <!-- Selection Overlay (Optional visual cue) -->
                <div v-if="selected" class="position-absolute top-0 start-0 w-100 h-100 bg-primary opacity-25 rounded pointer-events-none"></div>
                <!-- Sender Name (Group Chat) -->
                <div v-if="!isMe" class="small fw-bold mb-1" style="color: #d63384;">
                    {{ message.senderName || 'User ' + message.senderId }}
                </div>

                <!-- Reply Quote Block -->
                <div v-if="message.replyTo" class="mb-2 p-1 border-start border-3 rounded bg-opacity-25 bg-secondary small" style="background: rgba(0,0,0,0.1);">
                    <div class="fw-bold">{{ message.replyToSender || 'Unknown' }}</div>
                    <div class="text-truncate">{{ message.replyToText || 'Original message not found' }}</div>
                </div>

                <!-- Forwarded Label -->
                <div v-if="message.isForwarded" class="mb-1 small fst-italic d-flex align-items-center gap-1" :class="{'text-white-50': isMe, 'text-muted': !isMe}">
                    <svg class="feather" style="width: 12px; height: 12px;"><use href="/feather-sprite-v4.29.0.svg#corner-down-right"/></svg>
                    Forwarded
                </div>

                <!-- Image Content -->
                <div v-if="message.photoSource" class="mb-1" @click.stop="$emit('view-image', '/images/' + message.photoSource)">
                    <img :src="'/images/' + message.photoSource" alt="Photo" class="img-fluid rounded" style="max-height: 200px; object-fit: cover; cursor: pointer;">
                </div>

                <!-- Text Content (Caption or Message) -->
                <p v-if="message.textContent" class="mb-0 text-break" style="line-height: 1.4;">{{ message.textContent }}</p>

                <!-- Reactions Display -->
                <div v-if="message.reactions && message.reactions.length > 0" class="d-flex flex-wrap gap-1 mt-1 mb-1">
                    <button v-for="r in message.reactions" :key="r.emoji" 
                        class="btn btn-sm py-0 px-1 border rounded-pill d-flex align-items-center gap-1"
                        :class="{'btn-primary bg-opacity-25 border-primary text-primary': r.isMe, 'btn-light border-secondary text-muted': !r.isMe}"
                        style="font-size: 0.8rem;"
                        title="Toggle Reaction"
                        @click="$emit('react', message, r.emoji)">
                        <span>{{ r.emoji }}</span>
                        <span class="small fw-bold">{{ r.count }}</span>
                    </button>
                    <!-- Info Button for Reactions -->
                    <button class="btn btn-sm btn-light py-0 px-1 border rounded-pill text-muted" title="Who reacted?" @click="$emit('view-reactions', message)">
                         <span style="font-size: 0.8rem;">?</span>
                    </button>
                </div>

                <!-- Timestamp & Checkmarks & Star -->
                <div class="d-flex justify-content-end align-items-center gap-1" style="margin-top: 2px;">
                    
                    <!-- Star Indicator -->
                    <svg v-if="message.isStarred" class="feather text-warning me-1" style="width: 12px; height: 12px; fill: currentColor;"><use href="/feather-sprite-v4.29.0.svg#star"/></svg>

                    <small class="text-end" :class="{'text-white-50': isMe, 'text-muted': !isMe}" style="font-size: 0.7rem;">
                        {{ formatTime(message.messageTimestamp) }}
                    </small>
                    
                    <!-- Status Icons (only for me) -->
                    <span v-if="isMe" style="font-size: 0.8rem;" :title="message.status">
                        <!-- Sent: Single Light Check (White-50 for contrast on Green) -->
                        <svg v-if="message.status === 'sent' || message.status === 'received'" class="feather text-white-50" style="width: 14px; height: 14px;"><use href="/feather-sprite-v4.29.0.svg#check"/></svg>
                        
                        <!-- Read: Double Cyan Check (Good contrast on Green) -->
                        <div v-if="message.status === 'read'" class="d-inline-flex position-relative" style="width: 18px;">
                             <svg class="feather text-info" style="width: 14px; height: 14px; position: absolute; left: 0;"><use href="/feather-sprite-v4.29.0.svg#check"/></svg>
                             <svg class="feather text-info" style="width: 14px; height: 14px; position: absolute; left: 5px;"><use href="/feather-sprite-v4.29.0.svg#check"/></svg>
                        </div>
                    </span>
                </div>
            </div>
        </div>
    </div>
</template>

<style scoped>
/* Show action buttons on hover */
.group-message-container:hover .action-buttons {
    opacity: 1 !important;
    transition: opacity 0.2s ease-in-out;
}
/* Custom background for sent messages */
.bg-myself {
    background-color: var(--msg-me-bg) !important;
    color: var(--msg-me-text) !important;
}

/* Increase padding for comfortable reading but not too large */
.card-body.p-2 {
    padding: 0.5rem 0.8rem 0.6rem 0.8rem !important;
}
</style>
