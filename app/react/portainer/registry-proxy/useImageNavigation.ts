import { useParamsState } from '@/react/hooks/useParamState';

export type ImageSelection = {
  repository: string | null;
  tag: string | null;
};

export function useImageNavigation() {
  const [{ repository, tag }, setParams] = useParamsState<ImageSelection>(
    (params) => ({
      repository: params.repository ?? null,
      tag: params.tag ?? null,
    })
  );

  return [
    { repository, tag },
    {
      openProxy: () => setParams({ repository: undefined, tag: undefined }),
      openRepository: (repository: string) =>
        setParams({ repository, tag: undefined }),
      openTag: (repository: string, tag: string) =>
        setParams({ repository, tag }),
    },
  ] as const;
}
