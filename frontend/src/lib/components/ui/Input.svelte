<script lang="ts">
  import type { HTMLInputAttributes } from 'svelte/elements';
  import { cn } from '../../utils';

  interface Props extends HTMLInputAttributes {
    class?: string;
    value?: any;
  }

  let { class: className = '', value = $bindable(), ...rest }: Props = $props();
</script>

<input
  bind:value={value}
  class={cn('flex h-10 w-full rounded-md border border-slate-200 bg-white px-3 py-2 text-sm ring-offset-white placeholder:text-slate-400 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-950 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50', className)}
  {...rest}
/>
