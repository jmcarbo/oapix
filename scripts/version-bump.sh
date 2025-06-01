#!/bin/bash
set -euo pipefail

# Get the latest tag
LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
echo "Latest tag: $LATEST_TAG"

# Remove 'v' prefix if present
CURRENT_VERSION=${LATEST_TAG#v}

# Split version into components
IFS='.' read -r MAJOR MINOR PATCH <<< "$CURRENT_VERSION"

# Get commits since last tag
if [ "$LATEST_TAG" = "v0.0.0" ]; then
    COMMITS=$(git log --pretty=format:"%s" --no-merges)
else
    COMMITS=$(git log "${LATEST_TAG}..HEAD" --pretty=format:"%s" --no-merges)
fi

# Analyze commits for version bump type
BUMP_TYPE="patch"
while IFS= read -r commit; do
    # Check for breaking changes (major bump)
    if echo "$commit" | grep -qE "^(feat|fix|refactor|perf|style|docs|test|chore)(\(.+\))?!:"; then
        BUMP_TYPE="major"
        break
    elif echo "$commit" | grep -qE "BREAKING CHANGE|BREAKING-CHANGE" || \
         echo "$commit" | grep -qE "^break(ing)?(\(.+\))?:"; then
        BUMP_TYPE="major"
        break
    # Check for features (minor bump)
    elif echo "$commit" | grep -qE "^feat(\(.+\))?:"; then
        if [ "$BUMP_TYPE" != "major" ]; then
            BUMP_TYPE="minor"
        fi
    fi
done <<< "$COMMITS"

# Apply version bump
case $BUMP_TYPE in
    major)
        MAJOR=$((MAJOR + 1))
        MINOR=0
        PATCH=0
        ;;
    minor)
        MINOR=$((MINOR + 1))
        PATCH=0
        ;;
    patch)
        PATCH=$((PATCH + 1))
        ;;
esac

NEW_VERSION="v${MAJOR}.${MINOR}.${PATCH}"

# Check if we're in dry-run mode
if [ "${DRY_RUN:-false}" = "true" ]; then
    echo "Dry run mode - would create tag: $NEW_VERSION"
    echo "Bump type: $BUMP_TYPE"
    echo ""
    echo "Commits since $LATEST_TAG:"
    echo "$COMMITS"
    exit 0
fi

# Create annotated tag
echo "Creating tag: $NEW_VERSION (bump type: $BUMP_TYPE)"

# Generate tag message with commit summary
TAG_MESSAGE="Release $NEW_VERSION

Changes since $LATEST_TAG:
"

# Add commit messages to tag
while IFS= read -r commit; do
    TAG_MESSAGE="${TAG_MESSAGE}
- $commit"
done <<< "$COMMITS"

# Create the tag
git tag -a "$NEW_VERSION" -m "$TAG_MESSAGE"

echo "Tag $NEW_VERSION created successfully"
echo "To push the tag, run: git push origin $NEW_VERSION"