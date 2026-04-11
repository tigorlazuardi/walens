<script lang="ts">
  import type { HTMLAttributes } from 'svelte/elements';
  import { cn } from '../../utils';

  interface Props extends HTMLAttributes<HTMLDivElement> {
    orientation?: 'horizontal' | 'vertical';
  }

  let { class: className = '', orientation = 'horizontal', ...rest }: Props = $props();
</script>

<div
  role="separator"
  aria-orientation={orientation}
  class={cn(orientation === 'horizontal' ? 'h-px w-full' : 'w-px h-full', 'shrink-0 bg-slate-200', className)}
  {...rest}></div>
