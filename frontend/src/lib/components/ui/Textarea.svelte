<script lang="ts">
  import type { HTMLTextareaAttributes } from 'svelte/elements';
  import { cn } from '../../utils';

  interface Props extends HTMLTextareaAttributes {
    class?: string;
    value?: any;
  }

  let { class: className = '', value = $bindable(), ...rest }: Props = $props();
</script>

<textarea
  bind:value={value}
  class={cn('flex min-h-[80px] w-full rounded-md border border-slate-200 bg-white px-3 py-2 text-sm ring-offset-white placeholder:text-slate-400 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-950 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50', className)}
  {...rest}></textarea>
