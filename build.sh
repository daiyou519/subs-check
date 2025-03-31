#!/bin/bash

# Exit on error
set -e

# App information
APP_NAME="bestsub"
BUILT_DATA="$(TZ='Asia/Shanghai' date +'%F %T %z')"
GIT_AUTHOR="bestrui"
GIT_VERSION=$(git describe --tags --abbrev=0)

# Build flags
LDFLAGS="-X 'main.Version=${GIT_VERSION}' \
         -X 'main.BuildTime=${BUILT_DATA}' \
         -X 'main.Author=${GIT_AUTHOR}' \
         -s -w"
MUSL_FLAGS="--extldflags '-static -fpic' ${LDFLAGS}"

# Build tools
MUSL_BASE="https://musl.cc/"
ANDROID_NDK_BASE="https://dl.google.com/android/repository/"
ANDROID_NDK_VERSION="r27c"
TOOLCHAIN_DIR="$HOME/.bestsub/toolchains"


# Prepare build environment
prepare_build() {
    echo "Preparing build environment..."
    mkdir -p "dist"
}


# Get MUSL architecture name
setup_musl_toolchain() {
    local arch=$1
    local musl_arch
    local download_url

    case $arch in
        "amd64")
            musl_arch="x86_64"
            download_url="${MUSL_BASE}${musl_arch}-linux-musl-cross.tgz"
            ;;
        "arm64")
            musl_arch="aarch64"
            download_url="${MUSL_BASE}${musl_arch}-linux-musl-cross.tgz"
            ;;
        "386")  
            musl_arch="i686"
            download_url="${MUSL_BASE}${musl_arch}-linux-musl-cross.tgz"
            ;;
        "arm-7")
            musl_arch="armv7l"
            download_url="${MUSL_BASE}${musl_arch}-linux-musleabihf-cross.tgz"
            ;;
        *)
            echo "Unsupported architecture: $arch"
            return 1
            ;;
    esac
    
    echo "Setting up MUSL toolchain for $arch..."
    if [ ! -d "$TOOLCHAIN_DIR/musl/${musl_arch}/bin" ]; then
        mkdir -p "$TOOLCHAIN_DIR/musl/${musl_arch}"
        echo "Downloading ${download_url}..."
        curl -s -L -o /tmp/${musl_arch}-linux-musl.tgz "${download_url}" > /dev/null 2>&1
        echo "Extracting ${musl_arch}-linux-musl.tgz..."
        sudo tar xf /tmp/${musl_arch}-linux-musl.tgz --strip-components 1 -C "$TOOLCHAIN_DIR/musl/${musl_arch}"
        rm /tmp/${musl_arch}-linux-musl.tgz
        echo "MUSL toolchain for $arch setup completed"
    else
        echo "MUSL toolchain for $arch already exists"
    fi
}

setup_android_toolchain() {
    if [ ! -d "$TOOLCHAIN_DIR/android-ndk" ]; then
        echo "Downloading Android NDK ${ANDROID_NDK_VERSION}..."
        curl -L -o /tmp/android-ndk-${ANDROID_NDK_VERSION}.zip "${ANDROID_NDK_BASE}android-ndk-${ANDROID_NDK_VERSION}-linux.zip" > /dev/null 2>&1
        mkdir -p "$TOOLCHAIN_DIR/android-ndk"
        echo "Extracting android-ndk-${ANDROID_NDK_VERSION}.zip..."
        sudo unzip -q /tmp/android-ndk-${ANDROID_NDK_VERSION}.zip -d "$TOOLCHAIN_DIR/android-ndk"
        rm /tmp/android-ndk-${ANDROID_NDK_VERSION}.zip
        echo "Android NDK ${ANDROID_NDK_VERSION} setup completed"
    else
        echo "Android NDK ${ANDROID_NDK_VERSION} already exists"
    fi
}

setup_linux_toolchain() {
    local arch=$1
    local toolchain_name
    local toolchain_cmd
    case $arch in
        "x86_64")
            toolchain_name="gcc"
            toolchain_cmd="gcc"
            ;;
        "arm64")
            toolchain_name="gcc-aarch64-linux-gnu"
            toolchain_cmd="aarch64-linux-gnu-gcc"
            ;;
        "x86")
            toolchain_name="gcc-i686-linux-gnu"
            toolchain_cmd="i686-linux-gnu-gcc"
            ;;
        "arm")
            toolchain_name="gcc-arm-linux-gnueabihf"
            toolchain_cmd="arm-linux-gnueabihf-gcc"
            ;;
        *)
            echo "Unsupported architecture: $arch"
            return 1
            ;;
    esac

    if ! command -v ${toolchain_cmd} &> /dev/null; then
        echo "Installing ${toolchain_name}..."
        sudo apt install -y ${toolchain_name}
    else
        echo "${toolchain_name} is already installed"
    fi
}

setup_xgo() {
    if ! command -v xgo &> /dev/null; then
        echo "Installing xgo..."
        go install github.com/crazy-max/xgo@latest
    else
        echo "xgo is already installed"
    fi
    if ! docker image inspect ghcr.io/crazy-max/xgo &> /dev/null; then
        echo "Pulling xgo image..."
        docker pull ghcr.io/crazy-max/xgo:latest
    else
        echo "xgo image is already pulled"
    fi
}

# Build for standard Linux
build_linux() {
    local arch=$1
    local cgo_cc
    local cgo_arch

    case $arch in
        "x86_64")
            cgo_arch=amd64
            cgo_cc="gcc"
            ;;
        "arm64")
            cgo_arch=arm64
            cgo_cc="aarch64-linux-gnu-gcc"
            ;;
        "x86")
            cgo_arch=386
            cgo_cc="i686-linux-gnu-gcc"
            ;;
        "arm")
            cgo_arch=arm
            cgo_cc="arm-linux-gnueabihf-gcc"
            ;;
        *)
            echo "Unsupported architecture: $arch"
            return 1
            ;;
    esac

    echo "Building Linux ${arch}..."
    GOOS=linux GOARCH=${cgo_arch} CGO_ENABLED=1 CC=${cgo_cc} \
        go build -o "./dist/${APP_NAME}-linux-${arch}" -ldflags="${LDFLAGS}" -tags=jsoniter ./cmd/server
    echo "Linux ${arch} build completed"
}

# Build for Linux MUSL
build_linux_musl() {
    local arch=$1
    local cgo_cc
    local cgo_arch

    case $arch in
        "amd64")
            cgo_arch=amd64
            cgo_cc="${TOOLCHAIN_DIR}/musl/x86_64/bin/x86_64-linux-musl-gcc"
            ;;
        "arm64")
            cgo_arch=arm64
            cgo_cc="${TOOLCHAIN_DIR}/musl/aarch64/bin/aarch64-linux-musl-gcc"
            ;;
        "386")
            cgo_arch=386
            cgo_cc="${TOOLCHAIN_DIR}/musl/i686/bin/i686-linux-musl-gcc"
            ;;
        "arm-7")
            cgo_arch=arm
            cgo_cc="${TOOLCHAIN_DIR}/musl/armv7l/bin/armv7l-linux-musleabihf-gcc"
            ;;
        *)
            echo "Unsupported architecture: $arch"
            return 1
            ;;
    esac

    echo "Building Linux MUSL ${arch}..."
    GOOS=linux GOARCH=${cgo_arch} CGO_ENABLED=1 CC=${cgo_cc} \
        go build -o "./dist/${APP_NAME}-linux-musl-${arch}" -ldflags="${MUSL_FLAGS}" -tags=jsoniter ./cmd/server
    echo "Linux MUSL ${arch} build completed"
}

build_android() {
    bin_path="${TOOLCHAIN_DIR}/android-ndk/android-ndk-${ANDROID_NDK_VERSION}/toolchains/llvm/prebuilt/linux-x86_64/bin"
    local arch=$1
    local cgo_cc
    local cgo_arch
    
    case $arch in
        "amd64")
            cgo_cc="x86_64-linux-android24-clang"
            cgo_arch="amd64"
            ;;
        "arm64")
            cgo_cc="aarch64-linux-android24-clang"
            cgo_arch="arm64"
            ;;
        "arm-7")
            cgo_cc="armv7a-linux-androideabi24-clang"
            cgo_arch="arm"
            ;;
        "386")
            cgo_cc="i686-linux-android24-clang"
            cgo_arch="386"
            ;;
        *)
            echo "Unsupported architecture: $1"
            return 1
            ;;
    esac
    echo "Building android ${cgo_arch}..."
    GOOS=android GOARCH=${cgo_arch} CC=${bin_path}/${cgo_cc} CGO_ENABLED=1 \
        go build -o "./dist/${APP_NAME}-android-${arch}" -ldflags="${LDFLAGS}" -tags=jsoniter ./cmd/server
    ${bin_path}/llvm-strip "./dist/${APP_NAME}-android-${arch}"
    echo "Android ${arch} build completed"
}

build_use_xgo() {
    local os=$1
    local arch=$2

    echo "Building for ${os} ${arch}..."
    if ! xgo -targets=${os}/${arch} -out "dist/${APP_NAME}" -ldflags="${LDFLAGS}" -tags=jsoniter -pkg ./cmd/server .; then
        echo "Failed to build for ${os} ${arch}"
        return 1
    fi
    echo "Build completed successfully!"
}

# Compress built files
compress_files() {
    echo "Compressing built files..."
    cd dist
    
    cp ../README.md ../LICENSE ./
    
    echo "Compressing Linux and Darwin builds..."
    for file in ${APP_NAME}-linux-* ${APP_NAME}-darwin-* ${APP_NAME}-android-*; do
        if [ -f "$file" ]; then
            cp "$file" "${APP_NAME}"
            tar -czf "${file}.tar.gz" "${APP_NAME}" README.md LICENSE
            rm "${APP_NAME}"   
            rm "$file"
        fi
    done
    
    echo "Compressing Windows builds..."
    for file in ${APP_NAME}-windows-*; do
        if [ -f "$file" ]; then
            cp "$file" "${APP_NAME}.exe"
            zip "${file%.*}.zip" "${APP_NAME}.exe" README.md LICENSE
            rm "${APP_NAME}.exe"   
            rm "$file"
        fi
    done
    
    rm README.md LICENSE
    
    cd ..
}

generate_checksums() {
    echo "Generating MD5 checksums..."
    cd dist
    find . -type f -print0 | xargs -0 md5sum > md5.txt
    cat md5.txt
    cd ..
}

rename_files() {
    echo "Renaming built files..."
    cd dist
    
    local renames=(
        "amd64:x86_64"
        "386:x86"
        "arm-7:armv7"
    )
    
    local max_length=0
    for file in ${APP_NAME}-*; do
        [ ${#file} -gt $max_length ] && max_length=${#file}
    done
    
    for file in ${APP_NAME}-*; do
        if [ -f "$file" ]; then
            local new_name="$file"
            for rename in "${renames[@]}"; do
                local old_arch="${rename%%:*}"
                local new_arch="${rename#*:}"
                new_name="${new_name//-$old_arch/-$new_arch}"
            done
            
            if [ "$file" != "$new_name" ]; then
                printf "%-${max_length}s -> %s\n" "$file" "$new_name"
                mv "$file" "$new_name"
            fi
        fi
    done
    
    cd ..
}

copy_docker_bin() {
    echo "Copying binaries for Docker build..."
    mkdir -p docker/{glibc,musl}
    
    local platforms=(
        "amd64:linux/amd64"
        "386:linux/386"
        "arm-7:linux/arm/v7"
        "arm64:linux/arm64"
    )
    
    local max_length=0
    for platform in "${platforms[@]}"; do
        local arch="${platform%%:*}"
        local name_glibc="${APP_NAME}-linux-${arch}"
        local name_musl="${APP_NAME}-linux-musl-${arch}"
        
        [ ${#name_glibc} -gt $max_length ] && max_length=${#name_glibc}
        [ ${#name_musl} -gt $max_length ] && max_length=${#name_musl}
    done
    
    for platform in "${platforms[@]}"; do
        local arch="${platform%%:*}"
        local docker_platform="${platform#*:}"
        
        mkdir -p "docker/glibc/${docker_platform}"
        mkdir -p "docker/musl/${docker_platform}"
        
        local name_glibc="${APP_NAME}-linux-${arch}"
        if [ -f "dist/${name_glibc}" ]; then
            printf "%-${max_length}s -> docker/glibc/%s/%s\n" \
                "${name_glibc}" "${docker_platform}" "${APP_NAME}"
            cp "dist/${name_glibc}" "docker/glibc/${docker_platform}/${APP_NAME}"
        elif [ -f "dist/${name_glibc}.tar.gz" ]; then
            printf "%-${max_length}s -> docker/glibc/%s/%s (from tar)\n" \
                "${name_glibc}" "${docker_platform}" "${APP_NAME}"
            tar xzf "dist/${name_glibc}.tar.gz" -C "docker/glibc/${docker_platform}" "${APP_NAME}"
        else
            printf "%-${max_length}s -> not found\n" "${name_glibc}"
        fi
        
        local name_musl="${APP_NAME}-linux-musl-${arch}"
        if [ -f "dist/${name_musl}" ]; then
            printf "%-${max_length}s -> docker/musl/%s/%s\n" \
                "${name_musl}" "${docker_platform}" "${APP_NAME}"
            cp "dist/${name_musl}" "docker/musl/${docker_platform}/${APP_NAME}"
        elif [ -f "dist/${name_musl}.tar.gz" ]; then
            printf "%-${max_length}s -> docker/musl/%s/%s (from tar)\n" \
                "${name_musl}" "${docker_platform}" "${APP_NAME}"
            tar xzf "dist/${name_musl}.tar.gz" -C "docker/musl/${docker_platform}" "${APP_NAME}"
        else
            printf "%-${max_length}s -> not found\n" "${name_musl}"
        fi
    done
    
    echo "Docker binaries prepared successfully"
}


if [ "$1" == "release" ]; then
    
    prepare_build
    setup_android_toolchain
    setup_musl_toolchain arm-7
    setup_musl_toolchain 386
    setup_musl_toolchain arm64
    setup_musl_toolchain amd64
    setup_xgo

    build_android amd64
    build_android 386
    build_android arm-7
    build_android arm64

    build_linux_musl arm-7
    build_linux_musl arm64
    build_linux_musl amd64
    build_linux_musl 386

    build_use_xgo linux amd64
    build_use_xgo linux 386
    build_use_xgo linux arm64
    build_use_xgo linux arm-7

    build_use_xgo windows amd64
    build_use_xgo windows 386
    build_use_xgo windows amd64

    build_use_xgo darwin arm64
    build_use_xgo darwin amd64

    copy_docker_bin
    rename_files
    generate_checksums
    compress_files

fi
