<script lang="ts">
  import type { HTMLAttributes } from 'svelte/elements';
  import type { Snippet } from 'svelte';
  import { cn } from '../../utils';

  type Variant = 'default' | 'secondary' | 'success' | 'warning' | 'destructive' | 'outline';

  interface Props extends HTMLAttributes<HTMLSpanElement> {
    variant?: Variant;
    children?: Snippet;
  }

  let { class: className = '', variant = 'default', children, ...rest }: Props = $props();

  const variants: Record<Variant, string> = {
    default: 'border-transparent bg-slate-900 text-white',
    secondary: 'border-transparent bg-slate-100 text-slate-900',
    success: 'border-transparent bg-emerald-100 text-emerald-800',
    warning: 'border-transparent bg-amber-100 text-amber-800',
    destructive: 'border-transparent bg-rose-100 text-rose-800',
    outline: 'text-slate-700 border-slate-200 bg-white',
  };
</script>

<span
  class={cn('inline-flex items-center rounded-md border px-2.5 py-0.5 text-xs font-semibold transition-colors', variants[variant], className)}
  {...rest}
>
  {@render children?.()}
</span>
