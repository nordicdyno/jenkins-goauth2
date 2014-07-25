#!/bin/sh
#default_make_bash=$(go env GOROOT)
#default_make_bash="$default_make_bash/src/make.bash"
#makedir=${makebash:-"$default_make_bash"}
version="v1.1"
os=$(go env GOOS)
arch=$(go env GOARCH)
goversion=$(go version | awk '{print $3}')

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

mkdir -p dist
for os in linux darwin windows; do
    GOARCH=$arch
    SUFFIX=
    if [ "$os" == "windows" ]; then
        GOARCH=386
        SUFFIX=".exe"
    fi
    echo "... building v$version for $os/$GOARCH"
    BUILD=$(mkdir -p _build)
    TARGET="jenkins-goauth2.$version.$os-$GOARCH.$goversion"
    pushd _build
    GOOS=$os GOARCH=$GOARCH go build github.com/nordicdyno/jenkins-goauth2 \
        || exit 1
    #exit
    tar czvf $TARGET.tar.gz jenkins-goauth2$SUFFIX
    mv $TARGET.tar.gz $DIR/dist
    popd
    rm -rf _build
    #make clean
done
