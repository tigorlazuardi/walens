<script lang="ts">
  import type { HTMLButtonAttributes } from 'svelte/elements';
  import type { Snippet } from 'svelte';
  import { cn } from '../../utils';

  type Variant = 'default' | 'secondary' | 'outline' | 'ghost' | 'destructive';
  type Size = 'sm' | 'default' | 'lg' | 'icon';

  interface Props extends HTMLButtonAttributes {
    variant?: Variant;
    size?: Size;
    children?: Snippet;
  }

  let { class: className = '', variant = 'default', size = 'default', children, ...rest }: Props = $props();

  const variants: Record<Variant, string> = {
    default: 'bg-slate-900 text-white hover:bg-slate-800',
    secondary: 'bg-slate-100 text-slate-900 hover:bg-slate-200',
    outline: 'border border-slate-200 bg-white text-slate-900 hover:bg-slate-50',
    ghost: 'bg-transparent text-slate-900 hover:bg-slate-100',
    destructive: 'bg-rose-600 text-white hover:bg-rose-700',
  };

  const sizes: Record<Size, string> = {
    sm: 'h-8 px-3 text-sm',
    default: 'h-10 px-4 py-2',
    lg: 'h-11 px-6',
    icon: 'h-10 w-10',
  };
</script>

<button
  class={cn(
    'inline-flex items-center justify-center rounded-md border border-transparent font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-950 focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50',
    variants[variant],
    sizes[size],
    className,
  )}
  {...rest}
>
  {@render children?.()}
</button>
