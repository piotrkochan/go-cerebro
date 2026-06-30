import type { ButtonHTMLAttributes, ReactNode } from 'react';

import { Icon, type IconName } from './Icon';

type ButtonVariant = 'default' | 'primary' | 'success' | 'warning' | 'danger' | 'info';
type ButtonSize = 'xs' | 'sm' | 'md';

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  icon?: IconName;
  spin?: boolean;
  size?: ButtonSize;
  variant?: ButtonVariant;
  children: ReactNode;
};

export function Button({
  children,
  className = '',
  icon,
  size = 'md',
  spin,
  type = 'button',
  variant = 'default',
  ...props
}: ButtonProps) {
  const sizeClass = size === 'md' ? '' : `btn-${size}`;
  const classes = ['btn', `btn-${variant}`, sizeClass, className].filter(Boolean).join(' ');
  return (
    <button className={classes} type={type} {...props}>
      {icon ? <Icon name={icon} spin={spin} /> : null} {children}
    </button>
  );
}
