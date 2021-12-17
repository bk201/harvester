```
export DRONE_TAG=1.0.0-rc2

# Need a private registry and docker needs to trust it
export REGISTRY=10.10.0.1:5000

# or push to your own dockerhub. Remove REGISTRY exports and
# export REPOSITORY_ORG=bk201z

git clone https://github.com/harvester/harvester.git
cd harvester

# This is my own script, basically it pulls branch of a PR and merge it
# Use your preferred way to checkout a PR.
# You can also use Github CLI: gh
kf-review-pr.sh https://github.com/harvester/harvester/pull/1688

git remote add kf https://github.com/bk201/harvester.git
git fetch kf
git cherry-pick kf/test-release

./scripts/build-dev
```
