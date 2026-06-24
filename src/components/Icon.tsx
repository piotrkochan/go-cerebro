import {
  AlignLeft,
  ArrowDownAZ,
  ArrowDownZA,
  BookOpen,
  Camera,
  Check,
  ChevronDown,
  Circle,
  Clipboard,
  Code2,
  Crop,
  Crosshair,
  Database,
  Download,
  Edit3,
  File,
  Folder,
  FolderOpen,
  Hammer,
  HardDrive,
  History,
  Info,
  LoaderCircle,
  Lock,
  Maximize2,
  Minimize2,
  Move,
  Network,
  Pencil,
  Plug,
  Plus,
  Puzzle,
  RefreshCw,
  Save,
  Server,
  Settings,
  Settings2,
  Sparkles,
  Star,
  Tag,
  Tags,
  Trash2,
  TriangleAlert,
  Undo2,
  Unlock,
  WandSparkles,
  Wrench,
  Zap,
  type LucideIcon,
} from 'lucide-react';
import type { ComponentProps } from 'react';

export type IconName =
  | 'align-left'
  | 'bolt'
  | 'book'
  | 'camera'
  | 'caret-down'
  | 'check'
  | 'circle'
  | 'clipboard'
  | 'code'
  | 'cog'
  | 'cogs'
  | 'compress'
  | 'crop'
  | 'crosshairs'
  | 'database'
  | 'download'
  | 'edit'
  | 'expand'
  | 'file'
  | 'folder'
  | 'folder-open'
  | 'gavel'
  | 'hdd'
  | 'history'
  | 'info'
  | 'list'
  | 'lock'
  | 'magic'
  | 'pencil'
  | 'plug'
  | 'plus'
  | 'puzzle'
  | 'refresh'
  | 'save'
  | 'server'
  | 'sitemap'
  | 'sort-alpha-asc'
  | 'sort-alpha-desc'
  | 'spinner'
  | 'star'
  | 'star-o'
  | 'tag'
  | 'tags'
  | 'trash'
  | 'undo'
  | 'unlock'
  | 'warning'
  | 'wrench'
  | 'arrows';

const icons: Record<IconName, LucideIcon> = {
  'align-left': AlignLeft,
  arrows: Move,
  bolt: Zap,
  book: BookOpen,
  camera: Camera,
  'caret-down': ChevronDown,
  check: Check,
  circle: Circle,
  clipboard: Clipboard,
  code: Code2,
  cog: Settings,
  cogs: Settings2,
  compress: Minimize2,
  crop: Crop,
  crosshairs: Crosshair,
  database: Database,
  download: Download,
  edit: Edit3,
  expand: Maximize2,
  file: File,
  folder: Folder,
  'folder-open': FolderOpen,
  gavel: Hammer,
  hdd: HardDrive,
  history: History,
  info: Info,
  list: AlignLeft,
  lock: Lock,
  magic: WandSparkles,
  pencil: Pencil,
  plug: Plug,
  plus: Plus,
  puzzle: Puzzle,
  refresh: RefreshCw,
  save: Save,
  server: Server,
  sitemap: Network,
  'sort-alpha-asc': ArrowDownAZ,
  'sort-alpha-desc': ArrowDownZA,
  spinner: LoaderCircle,
  star: Star,
  'star-o': Star,
  tag: Tag,
  tags: Tags,
  trash: Trash2,
  undo: Undo2,
  unlock: Unlock,
  warning: TriangleAlert,
  wrench: Wrench,
};

type IconProps = Omit<ComponentProps<LucideIcon>, 'ref'> & {
  name: IconName;
  spin?: boolean;
  title?: string;
};

export function Icon({ className = '', name, size = 14, spin, title, ...props }: IconProps) {
  const Component = icons[name] ?? Sparkles;
  const classes = ['lucide-icon', spin ? 'lucide-spin' : '', name === 'star' ? 'lucide-fill' : '', className]
    .filter(Boolean)
    .join(' ');
  return (
    <Component aria-hidden={title ? undefined : true} aria-label={title} className={classes} focusable="false" size={size} {...props}>
      {title ? <title>{title}</title> : null}
    </Component>
  );
}
