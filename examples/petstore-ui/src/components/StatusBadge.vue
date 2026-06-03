<template>
	<Badge :variant="variant">{{ status }}</Badge>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { Badge } from '@fromforgesoftware/vue-kit';

// Maps both pet and adoption lifecycle statuses to a badge tone.
const props = defineProps<{ status: string }>();

const variant = computed(() => {
	switch (props.status) {
		case 'AVAILABLE':
		case 'COMPLETED':
			return 'success' as const;
		case 'PENDING':
		case 'PLACED':
			return 'warning' as const;
		case 'ADOPTED':
			return 'secondary' as const;
		case 'FAILED':
			return 'destructive' as const;
		default:
			return 'outline' as const;
	}
});
</script>
