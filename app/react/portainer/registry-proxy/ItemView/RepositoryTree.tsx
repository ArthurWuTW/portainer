import { useMemo, useState } from 'react';
import { Folder, FolderOpen, Package } from 'lucide-react';
import clsx from 'clsx';

import { CollapseExpandButton } from '@@/CollapseExpandButton';

const INDENT_PX = 16;

interface RepositoryNode {
  name: string;
  path: string;
  isRepository: boolean;
  children: RepositoryNode[];
}

interface Props {
  repositories: string[];
  selectedRepository: string | null;
  onSelectRepository: (repository: string) => void;
}

export function RepositoryTree({
  repositories,
  selectedRepository,
  onSelectRepository,
}: Props) {
  const nodes = useMemo(
    () => buildRepositoryTree(repositories),
    [repositories]
  );

  const [expandedPaths, setExpandedPaths] = useState<Set<string>>(
    () => new Set(ancestorPaths(selectedRepository))
  );

  function togglePath(path: string) {
    setExpandedPaths((previous) => {
      const next = new Set(previous);
      if (next.has(path)) {
        next.delete(path);
      } else {
        next.add(path);
      }
      return next;
    });
  }

  function selectRepository(path: string) {
    onSelectRepository(path);
    setExpandedPaths(
      (previous) => new Set([...previous, path, ...ancestorPaths(path)])
    );
  }

  return (
    <div
      className="border-border max-h-[420px] overflow-auto rounded border-2 border-solid py-1"
      role="tree"
      aria-label="Image repositories"
      data-cy="registry-proxy-repository-tree"
    >
      {nodes.map((node) => (
        <TreeRow
          key={node.path}
          node={node}
          depth={0}
          expandedPaths={expandedPaths}
          selectedRepository={selectedRepository}
          onTogglePath={togglePath}
          onSelectRepository={selectRepository}
        />
      ))}
    </div>
  );
}

interface TreeRowProps {
  node: RepositoryNode;
  depth: number;
  expandedPaths: Set<string>;
  selectedRepository: string | null;
  onTogglePath: (path: string) => void;
  onSelectRepository: (path: string) => void;
}

function TreeRow({
  node,
  depth,
  expandedPaths,
  selectedRepository,
  onTogglePath,
  onSelectRepository,
}: TreeRowProps) {
  const hasChildren = node.children.length > 0;
  const isExpanded = expandedPaths.has(node.path);
  const isSelected = node.isRepository && selectedRepository === node.path;

  function handleClick() {
    if (node.isRepository) {
      onSelectRepository(node.path);
    } else if (hasChildren) {
      onTogglePath(node.path);
    }
  }

  return (
    <>
      <div
        role="treeitem"
        tabIndex={0}
        aria-selected={isSelected}
        aria-expanded={hasChildren ? isExpanded : undefined}
        title={node.path}
        data-cy={`registry-proxy-repository-${node.path}`}
        className={clsx(
          'flex h-8 cursor-pointer select-none items-center gap-1 pr-2 hover:bg-gray-3 th-highcontrast:hover:bg-gray-iron-10 th-dark:hover:bg-gray-iron-9',
          isSelected && '!bg-[var(--primary-50)]'
        )}
        style={{ paddingLeft: depth * INDENT_PX + 12 }}
        onClick={handleClick}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            handleClick();
          }
        }}
      >
        {hasChildren ? (
          <CollapseExpandButton
            quarterRotation
            isExpanded={isExpanded}
            onClick={() => onTogglePath(node.path)}
          />
        ) : (
          <span className="w-[22px] shrink-0" />
        )}
        <NodeIcon node={node} isExpanded={isExpanded} />
        <span
          className="flex-1 truncate text-sm text-gray-11 th-highcontrast:text-white th-dark:text-white"
          title={node.path}
        >
          {node.name}
        </span>
      </div>

      {isExpanded &&
        node.children.map((child) => (
          <TreeRow
            key={child.path}
            node={child}
            depth={depth + 1}
            expandedPaths={expandedPaths}
            selectedRepository={selectedRepository}
            onTogglePath={onTogglePath}
            onSelectRepository={onSelectRepository}
          />
        ))}
    </>
  );
}

function NodeIcon({
  node,
  isExpanded,
}: {
  node: RepositoryNode;
  isExpanded: boolean;
}) {
  if (node.isRepository) {
    return (
      <Package
        size={14}
        className="shrink-0 text-gray-7 th-highcontrast:text-white th-dark:text-gray-5"
      />
    );
  }

  const FolderIcon = isExpanded ? FolderOpen : Folder;

  return (
    <FolderIcon
      size={14}
      className="shrink-0 text-blue-7 th-highcontrast:text-white th-dark:text-blue-5"
    />
  );
}

function buildRepositoryTree(repositories: string[]): RepositoryNode[] {
  const root: RepositoryNode = {
    name: '',
    path: '',
    isRepository: false,
    children: [],
  };

  for (const repository of repositories) {
    const parts = repository.split('/').filter((part) => part.length > 0);
    let current = root;

    parts.forEach((part, index) => {
      const path = parts.slice(0, index + 1).join('/');
      let child = current.children.find((node) => node.name === part);

      if (!child) {
        child = { name: part, path, isRepository: false, children: [] };
        current.children.push(child);
      }

      if (index === parts.length - 1) {
        child.isRepository = true;
      }

      current = child;
    });
  }

  return sortNodes(root.children);
}

function sortNodes(nodes: RepositoryNode[]): RepositoryNode[] {
  return nodes
    .toSorted((a, b) => {
      const aHasChildren = a.children.length > 0;
      const bHasChildren = b.children.length > 0;

      if (aHasChildren && !bHasChildren) return -1;
      if (!aHasChildren && bHasChildren) return 1;

      return a.name.localeCompare(b.name, undefined, { sensitivity: 'base' });
    })
    .map((node) => ({ ...node, children: sortNodes(node.children) }));
}

function ancestorPaths(path: string | null): string[] {
  if (!path) {
    return [];
  }

  const parts = path.split('/');

  return parts
    .slice(0, -1)
    .map((_, index) => parts.slice(0, index + 1).join('/'));
}
