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

  # Make sure we're using the latest version of Chef
  config.omnibus.chef_version = :latest

  config.vm.define "api", primary: true do |api|
    api.vm.synced_folder ".", "/usr/local/deckbrew"
    api.vm.synced_folder "~/projects/coiltap", "/usr/local/coiltap"
    api.vm.provision :chef_solo do |chef|

      chef.json = {
          "deckbrew" => {
              "database" => {
                "user" => ENV["DATABASE_USER"],
                "password" => ENV["DATABASE_PASSWORD"],
              },
              "coiltap" => "http://ec2-54-193-42-245.us-west-1.compute.amazonaws.com:9200/coiltap-dev",
              "hostname" => "http://localhost:6001",
              "event" => "vagrant-ready",
          }
      }

      chef.cookbooks_path = "cookbooks"
      chef.add_recipe "deckbrew::database"
    end
  end

  config.vm.define "image" do |image|
    image.vm.provision :chef_solo do |chef|
      chef.cookbooks_path = "cookbooks"
      chef.add_recipe "deckbrew::image"
    end
  end
end
