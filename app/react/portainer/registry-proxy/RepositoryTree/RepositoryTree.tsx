import { useMemo, useState } from 'react';
import { Folder, FolderOpen, Package, Tag } from 'lucide-react';
import clsx from 'clsx';

import { CollapseExpandButton } from '@@/CollapseExpandButton';

import { RegistryTag } from '../types';

const INDENT_PX = 16;

interface RepositoryNode {
  name: string;
  path: string;
  isRepository: boolean;
  tags: RegistryTag[];
  children: RepositoryNode[];
}

interface Selection {
  repository: string | null;
  tag: string | null;
}

interface Props {
  repositories: string[];
  selection: Selection;
  tags: RegistryTag[];
  isLoadingTags: boolean;
  onOpenRepository: (repository: string) => void;
  onOpenTag: (repository: string, tag: string) => void;
}

export function RepositoryTree({
  repositories,
  selection,
  tags,
  isLoadingTags,
  onOpenRepository,
  onOpenTag,
}: Props) {
  const nodes = useMemo(
    () => buildRepositoryTree(repositories, selection.repository, tags),
    [repositories, selection.repository, tags]
  );

  const [expandedPaths, setExpandedPaths] = useState<Set<string>>(
    () => new Set(initialExpandedPaths(selection))
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

  return (
    <div
      className="border-border max-h-[calc(100vh-260px)] min-h-[200px] overflow-auto rounded border-2 border-solid py-1"
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
          selection={selection}
          isLoadingTags={isLoadingTags}
          onTogglePath={togglePath}
          onOpenRepository={onOpenRepository}
          onOpenTag={onOpenTag}
        />
      ))}
    </div>
  );
}

interface TreeRowProps {
  node: RepositoryNode;
  depth: number;
  expandedPaths: Set<string>;
  selection: Selection;
  isLoadingTags: boolean;
  onTogglePath: (path: string) => void;
  onOpenRepository: (repository: string) => void;
  onOpenTag: (repository: string, tag: string) => void;
}

function TreeRow({
  node,
  depth,
  expandedPaths,
  selection,
  isLoadingTags,
  onTogglePath,
  onOpenRepository,
  onOpenTag,
}: TreeRowProps) {
  const isExpanded = expandedPaths.has(node.path);
  const isSelected =
    node.isRepository && selection.repository === node.path && !selection.tag;
  const hasTags = node.isRepository && node.path === selection.repository;
  const hasChildren = node.children.length > 0 || hasTags;

  function handleClick() {
    if (node.isRepository) {
      onOpenRepository(node.path);
      if (!isExpanded) {
        onTogglePath(node.path);
      }
    } else if (node.children.length > 0) {
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

      {isExpanded && (
        <>
          {node.children.map((child) => (
            <TreeRow
              key={child.path}
              node={child}
              depth={depth + 1}
              expandedPaths={expandedPaths}
              selection={selection}
              isLoadingTags={isLoadingTags}
              onTogglePath={onTogglePath}
              onOpenRepository={onOpenRepository}
              onOpenTag={onOpenTag}
            />
          ))}
          {hasTags && (
            <TagNodes
              repository={node.path}
              tags={node.tags}
              depth={depth + 1}
              selection={selection}
              isLoadingTags={isLoadingTags}
              onOpenTag={onOpenTag}
            />
          )}
        </>
      )}
    </>
  );
}

interface TagNodesProps {
  repository: string;
  tags: RegistryTag[];
  depth: number;
  selection: Selection;
  isLoadingTags: boolean;
  onOpenTag: (repository: string, tag: string) => void;
}

function TagNodes({
  repository,
  tags,
  depth,
  selection,
  isLoadingTags,
  onOpenTag,
}: TagNodesProps) {
  if (isLoadingTags && tags.length === 0) {
    return (
      <div
        className="flex h-8 items-center text-sm text-gray-7"
        style={{ paddingLeft: depth * INDENT_PX + 46 }}
      >
        Loading versions...
      </div>
    );
  }

  if (tags.length === 0) {
    return (
      <div
        className="flex h-8 items-center text-sm text-gray-7"
        style={{ paddingLeft: depth * INDENT_PX + 46 }}
      >
        No versions
      </div>
    );
  }

  return (
    <>
      {tags.map((tag) => (
        <TagRow
          key={tag.name}
          repository={repository}
          tag={tag}
          depth={depth}
          isSelected={
            selection.repository === repository && selection.tag === tag.name
          }
          onOpenTag={onOpenTag}
        />
      ))}
    </>
  );
}

interface TagRowProps {
  repository: string;
  tag: RegistryTag;
  depth: number;
  isSelected: boolean;
  onOpenTag: (repository: string, tag: string) => void;
}

function TagRow({
  repository,
  tag,
  depth,
  isSelected,
  onOpenTag,
}: TagRowProps) {
  return (
    <div
      role="treeitem"
      tabIndex={0}
      aria-selected={isSelected}
      title={`${repository}:${tag.name}`}
      data-cy={`registry-proxy-tag-${repository}-${tag.name}`}
      className={clsx(
        'flex h-8 cursor-pointer select-none items-center gap-1 pr-2 hover:bg-gray-3 th-highcontrast:hover:bg-gray-iron-10 th-dark:hover:bg-gray-iron-9',
        isSelected && '!bg-[var(--primary-50)]'
      )}
      style={{ paddingLeft: depth * INDENT_PX + 12 }}
      onClick={() => onOpenTag(repository, tag.name)}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault();
          onOpenTag(repository, tag.name);
        }
      }}
    >
      <span className="w-[22px] shrink-0" />
      <Tag
        size={14}
        className="shrink-0 text-gray-7 th-highcontrast:text-white th-dark:text-gray-5"
      />
      <span
        className="flex-1 truncate text-sm text-gray-11 th-highcontrast:text-white th-dark:text-white"
        title={`${repository}:${tag.name}`}
      >
        {tag.name}
      </span>
    </div>
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

function buildRepositoryTree(
  repositories: string[],
  selectedRepository: string | null,
  tags: RegistryTag[]
): RepositoryNode[] {
  const root: RepositoryNode = {
    name: '',
    path: '',
    isRepository: false,
    tags: [],
    children: [],
  };

  for (const repository of repositories) {
    const parts = repository.split('/').filter((part) => part.length > 0);
    let current = root;

    parts.forEach((part, index) => {
      const path = parts.slice(0, index + 1).join('/');
      let child = current.children.find((node) => node.name === part);

      if (!child) {
        child = {
          name: part,
          path,
          isRepository: false,
          tags: [],
          children: [],
        };
        current.children.push(child);
      }

      if (index === parts.length - 1) {
        child.isRepository = true;
        child.tags = repository === selectedRepository ? tags : [];
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

function initialExpandedPaths({ repository, tag }: Selection): string[] {
  if (!repository) {
    return [];
  }

  const parts = repository.split('/');
  const ancestors = parts
    .slice(0, -1)
    .map((_, index) => parts.slice(0, index + 1).join('/'));

  return tag ? [...ancestors, repository] : ancestors;
}
