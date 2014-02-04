# -*- mode: ruby -*-
# vi: set ft=ruby :

# Vagrantfile API/syntax version. Don't touch unless you know what you're doing!
VAGRANTFILE_API_VERSION = "2"

Vagrant.configure(VAGRANTFILE_API_VERSION) do |config|
  # All Vagrant configuration is done here. The most common configuration
  # options are documented and commented below. For a complete reference,
  # please see the online documentation at vagrantup.com.

  # Every Vagrant virtual environment requires a box to build off of.
  config.vm.box = "bento-precise64"
  config.vm.box_url = "http://opscode-vm-bento.s3.amazonaws.com/vagrant/virtualbox/opscode_ubuntu-12.04_chef-provisionerless.box"

  # Create a forwarded port mapping which allows access to a specific port
  # within the machine from a port on the host machine. In the example below,
  # accessing "localhost:8080" will access port 80 on the guest machine.
  config.vm.network :forwarded_port, guest: 80, host: 6001
  config.vm.network :forwarded_port, guest: 3000, host: 6002

  # Share an additional folder to the guest VM. The first argument is
  # the path on the host to the actual folder. The second argument is
  # the path on the guest to mount the folder. And the optional third
  # argument is a set of non-required options.
  config.vm.synced_folder ".", "/usr/local/deckbrew"
  
  # Make sure we're using the latest version of Chef
  config.omnibus.chef_version = :latest

  config.vm.provision :chef_solo do |chef|

    chef.json = {
        "deckbrew" => {
            "hostname" => "http://localhost:6001",
            "event" => "vagrant-ready",
        }
    }

    chef.cookbooks_path = "cookbooks"
    chef.add_recipe "deckbrew::database"
  end

end
