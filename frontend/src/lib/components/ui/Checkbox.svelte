<script lang="ts">
  import type { HTMLInputAttributes } from 'svelte/elements';
  import { cn } from '../../utils';

  interface Props extends HTMLInputAttributes {
    class?: string;
    checked?: any;
  }

  let { class: className = '', checked = $bindable(false), ...rest }: Props = $props();
</script>

<input
  type="checkbox"
  bind:checked={checked}
  class={cn('h-4 w-4 shrink-0 rounded border border-slate-300 text-slate-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-950 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50', className)}
  {...rest}
/>
