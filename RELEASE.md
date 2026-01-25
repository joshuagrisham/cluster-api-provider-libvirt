# Creating a new release

Currently the release process is manual. A workflow could be set up in the future. The steps below are documented just to help facilitate the manual process.

```sh
export CAPLV_VERSION=0.2.0

# Generate manifests
make manifests generate

# Build image locally
docker build \
  -t ghcr.io/joshuagrisham/cluster-api-provider-libvirt:$CAPLV_VERSION \
  .

# Build and push image
docker build --push \
  -t ghcr.io/joshuagrisham/cluster-api-provider-libvirt:$CAPLV_VERSION \
  .

# Tag and push as 0.2
docker tag ghcr.io/joshuagrisham/cluster-api-provider-libvirt:$CAPLV_VERSION ghcr.io/joshuagrisham/cluster-api-provider-libvirt:0.2
docker push ghcr.io/joshuagrisham/cluster-api-provider-libvirt:0.2

# Tag and push latest
docker tag ghcr.io/joshuagrisham/cluster-api-provider-libvirt:$CAPLV_VERSION ghcr.io/joshuagrisham/cluster-api-provider-libvirt:latest
docker push ghcr.io/joshuagrisham/cluster-api-provider-libvirt:latest

# Create infrastructure-components.yaml
kustomize build config/default > infrastructure-components.yaml
```

Finally, create a new release with tag matching `v$CAPLV_VERSION` and upload these files to the release:

- `infrastructure-components.yaml`
- `metadata.yaml`
- all of the `./templates/`

## Test locally without releasing

```sh
# Create a provider file structure for clusterctl to read:
mkdir -p /tmp/infrastructure-libvirt/v$CAPLV_VERSION/
cp infrastructure-components.yaml /tmp/infrastructure-libvirt/v$CAPLV_VERSION/
cp metadata.yaml /tmp/infrastructure-libvirt/v$CAPLV_VERSION/
cp templates/*.yaml /tmp/infrastructure-libvirt/v$CAPLV_VERSION/

# Update ~/.config/cluster-api/clusterctl.yaml url to:
echo /tmp/infrastructure-libvirt/v$CAPLV_VERSION/infrastructure-components.yaml
```

Note that this will still require the desired image to be pushed and available. You may wish to temporarily update the image tag in the local `infrastructure-components.yaml` file if you want to test with an image that is not tagged `latest`.
