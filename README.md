# terraform-provider-ytsaurus

## Installation from Terraform mirror
### Example layout
- **\$HOME/.terraformrc** - Terraform configuration file.
- **$HOME/tf** - a folder with Terraform files with your infrastructure configuration.

### How to configure ~/.terraformrc
```
 $ cat ~/.terraformrc
provider_installation {
  network_mirror {
    url = "https://terraform-provider.ytsaurus.tech"
    include = ["registry.terraform.io/ytsaurus/*"]
  }
  direct {
    exclude = ["registry.terraform.io/ytsaurus/*"]
  }

}
```
### How to download YTsaurus terraform-provider
Run from your Terroform configuration
```
 $ cd $HOME/tf
 $ terraform init

```

## Installation from source code
### Prerequirements

Below is a list of things you should have or configure:

* golang
* GOPATH environment variable
* Terraform binary
* YTsaurus main repo
* YTsaurus cluster for acceptance tests

### Example layout

- **\$HOME/.terraformrc** - Terraform configuration file.
- **\$HOME/go** - golang home, and GOPATH value.
- **\$HOME/go/src/terraform-provider-ytsaurus** - a folder with the provider's source code from github.com:ytsaurus/terraform-provider-ytsaurus.git.
- **$HOME/ytsaurus** - a folder with terraform-provider-ytsaurus checked out from github.com:ytsaurus/terraform-provider-ytsaurus.git.
- **$HOME/tf** - a folder with Terraform files with your infrastructure configuration.

### How to run tests

```
~/ytsaurus/yt/docker/local/run_local_cluster.sh
cd ~/go/src/terraform-provider-ytsaurus
make test
```

### How to install

```
cd ~/go/src/terraform-provider-ytsaurus
make install
```

### How to configure ~/.terraformrc
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
