# terraform-provider-ytsaurus

## Prerequirements

Below is a list of things you should have or configure:

* golang
* GOPATH environment variable
* terraform binary
* YTsaurus main repo
* YTsaurus cluster for acceptance tests 

### Example layout

- **\$HOME/.terraformrc** - terraform configuration file with dev_overrides.
- **\$HOME/go** - golang home, and GOPATH value.
- **\$HOME/go/src/terraform-provider-ytsaurus** - a folder with provider's source code from github.com:ytsaurus/terraform-provider-ytsaurus.git.
- **$HOME/ytsaurus** - a folder with terraform-provider-ytsaurus checked out from github.com:ytsaurus/terraform-provider-ytsaurus.git.
- **$HOME/tf** - a folder with terraform files with your infrastructure configuration.

## How to run tests

```
~/ytsaurus/yt/docker/local/run_local_cluster.sh
cd ~/go/src/terraform-provider-ytsaurus
make test
```

## How to install

```
cd ~/go/src/terraform-provider-ytsaurus
make install 
```

## How to configure
Add dev_overrides to ~/.terraformrc
```
 $ cat ~/.terraformrc
provider_installation {
  dev_overrides {
      "registry.terraform.io/ytsaurus/ytsaurus" = "<HOME>/go/bin"
  }
  direct {}
}
```

## Examples

Please see \$HOME/go/src/terraform-provider-ytsaurus/examples
