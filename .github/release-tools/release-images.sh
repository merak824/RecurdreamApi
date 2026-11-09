#!/usr/bin/env bash
set -euo pipefail
: "${RELEASE_VERSION:?}" "${RELEASE_SHA:?}" "${GITHUB_REPOSITORY:?}" "${RUNNER_TEMP:?}"
owner=${GITHUB_REPOSITORY%%/*}
# 镜像包名跟随仓库名（小写），这样包会自动链接到仓库、权限与可见性继承仓库设置。
repo=${GITHUB_REPOSITORY##*/}
repo=${repo,,}
# 镜像 tag 保留 git tag 的 v 前缀，保证 `git tag v1.2.3` 对应 `镜像:v1.2.3`，不再被削成 1.2.3。
image_tag="v${RELEASE_VERSION}"
registries=("ghcr.io/${owner,,}/${repo}")
if [[ ${SIMPLE_RELEASE:-false} != true && ${DOCKERHUB_USERNAME:-skip} != skip ]]; then
  registries+=("${DOCKERHUB_USERNAME}/${repo}")
fi
arches=(amd64 arm64)
if [[ ${SIMPLE_RELEASE:-false} == true ]]; then arches=(amd64); fi
for arch in "${arches[@]}"; do
  args=(--platform "linux/$arch" --file ".release-context/$arch/Dockerfile"
    --label "org.opencontainers.image.version=$RELEASE_VERSION"
    --label "org.opencontainers.image.revision=$RELEASE_SHA"
    --label "org.opencontainers.image.source=https://github.com/$GITHUB_REPOSITORY")
  for registry in "${registries[@]}"; do
    args+=(--tag "$registry:$image_tag-$arch")
    if [[ ${SIMPLE_RELEASE:-false} == true ]]; then
      args+=(--tag "$registry:$image_tag" --tag "$registry:latest")
    fi
  done
  if [[ ${DRY_RUN:-false} == true ]]; then
    args+=(--output "type=oci,dest=$RUNNER_TEMP/${repo}-$arch.oci.tar")
  else
    args+=(--push)
  fi
  docker buildx build "${args[@]}" ".release-context/$arch"
done
if [[ ${DRY_RUN:-false} != true && ${SIMPLE_RELEASE:-false} != true ]]; then
  major=${image_tag%%.*}
  minor=${image_tag#*.}; minor=${minor%%.*}
  for registry in "${registries[@]}"; do
    docker buildx imagetools create \
      --tag "$registry:$image_tag" --tag "$registry:latest" \
      --tag "$registry:$major.$minor" --tag "$registry:$major" \
      "$registry:$image_tag-amd64" "$registry:$image_tag-arm64"
  done
fi
