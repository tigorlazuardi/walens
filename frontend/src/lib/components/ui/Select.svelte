<script lang="ts">
  import type { HTMLSelectAttributes } from 'svelte/elements';
  import type { Snippet } from 'svelte';
  import { cn } from '../../utils';

  interface Props extends HTMLSelectAttributes {
    class?: string;
    value?: any;
    children?: Snippet;
  }

  let { class: className = '', value = $bindable(), children, ...rest }: Props = $props();
</script>

<select
  bind:value={value}
  class={cn('flex h-10 w-full rounded-md border border-slate-200 bg-white px-3 py-2 text-sm ring-offset-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-950 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50', className)}
  {...rest}
>
  {@render children?.()}
</select>
