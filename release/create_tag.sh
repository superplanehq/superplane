#!/bin/bash

# Pull all tags from the remote repository
git fetch --tags

# Get the latest semantic version tag
latest_tag=$(git tag -l --sort=-version:refname | head -n 1)

# Calc versions
PATCH=$(echo $latest_tag | awk -F. '{print $3 + 1}')
MINOR=$(echo $latest_tag | awk -F. '{print $2}')
MAJOR=$(echo $latest_tag | awk -F. '{print $1}')

# Create new tag
case $1 in
  patch)
    new_tag="$MAJOR.$MINOR.$PATCH"
    ;;
  minor)
    MINOR=$((MINOR + 1))
    new_tag="$MAJOR.$MINOR.0"
    ;;
  major)
    MAJOR=$((MAJOR + 1))
    new_tag="$MAJOR.0.0"
    ;;
  *)
    echo "Usage: $0 {patch|minor|major}"
    exit 1
    ;;
esac

echo "Creating new tag: $new_tag"
git tag $new_tag
git push origin $new_tag
