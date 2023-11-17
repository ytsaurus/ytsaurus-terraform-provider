#!/bin/bash

TF_META="{
  \"protocols\": [\"5.1\"],
  \"os\": \$os,
  \"arch\": \$arch,
  \"filename\": \$filename,
  \"download_url\": \$download_url,
  \"shasums_url\": \$shasums_url,
  \"shasums_signature_url\": \$shasums_signature_url,
  \"shasum\": \$shasum,
  \"signing_keys\": {
    \"gpg_public_keys\": [
      {
        \"key_id\": \"65C58BAFA065D15A61D5385B546AD1B32A4BD4ED\",
        \"ascii_armor\": \"-----BEGIN PGP PUBLIC KEY BLOCK-----\n\n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\nPSdrDg==\n=yKGA\n-----END PGP PUBLIC KEY BLOCK-----\"
      }
    ]
  }
}"


usage() {
  echo "usage: ./$0 VERSION"
}

create_tf_metadata() {
  version=$1
  os=$2
  archs=$3
  url=https://terraform-provider.ytsaurus.tech/ytsaurus/ytsaurus

  for arch in $archs; do
    filename=terraform-provider-ytsaurus_${version}_${os}_${arch}.zip
    shasum=$(shasum -a 256 tf_registry_metadata/$version/$filename|awk '{print $1}')
    mkdir -p tf_registry_metadata/$version/download/$os
    jq -n \
    --arg version "$version" \
    --arg os "$os" \
    --arg arch "$arch" \
    --arg filename "$filename" \
    --arg download_url "$url/$version/$filename" \
    --arg shasums_url "$url/$version/shasums" \
    --arg shasums_signature_url "$url/$version/shasums.sig" \
    --arg shasum "$shasum" \
    "$TF_META" > tf_registry_metadata/$version/download/$os/$arch
  done
}


build() {
  version=$1
  os=$2
  archs="$3"

  ext=""
  if [ "$os" == "windows" ]; then
    ext=".exe"
  fi

  for arch in $archs; do
    GOOS=$os GOARCH=$arch go build -o ./tf_registry_metadata/$version/$os/$arch/terraform-provider-ytsaurus${ext} .
    zip -j ./tf_registry_metadata/$version/terraform-provider-ytsaurus_${version}_${os}_${arch}.zip ./tf_registry_metadata/$version/$os/$arch/terraform-provider-ytsaurus${ext}
  done
  rm -rf ./tf_registry_metadata/$version/$os
  create_tf_metadata $1 $2 "$3"
}


version=$1
if [ -z "$version" ]; then
  echo "Error: version is empty" >&1
  usage
  exit 1
fi

test -d ./tf_registry_metadata/$version && rm -rf ./tf_registry_metadata/$version

build "$version" "linux" "amd64"
build "$version" "darwin" "amd64 arm64"
build "$version" "windows" "amd64"

cd ./tf_registry_metadata/$version
shasum -a 256 *.zip > shasums
gpg --output shasums.sig --detach-sig shasums
cd ../..
