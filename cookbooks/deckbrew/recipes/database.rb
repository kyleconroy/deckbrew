apt_repository 'pgdg' do
  uri          'http://apt.postgresql.org/pub/repos/apt/'
  distribution 'precise-pgdg'
  components   ['main']
  key          'https://www.postgresql.org/media/keys/ACCC4CF8.asc'
end

apt_repository 'varnish' do
  uri          'http://repo.varnish-cache.org/ubuntu/'
  distribution 'precise'
  components   ['varnish-3.0']
  key          'http://repo.varnish-cache.org/debian/GPG-key.txt'
end

package 'make'
package 'git'
package 'varnish'
package 'postgresql-9.3'

tar_extract 'https://go.googlecode.com/files/go1.2.linux-amd64.tar.gz' do
  target_dir '/usr/local'
  creates '/usr/local/go/bin'
end

template "go" do
  path "/etc/profile.d/go.sh"
  source "goprofile.erb"
  owner "root"
  group "root"
  mode "0755"
end

directory 'gopath'

# Build the binary
execute 'make deps' do
  cwd '/usr/local/deckbrew'
end

execute 'make' do
  cwd '/usr/local/deckbrew'
end

# Create the database
execute 'make syncdb' do
  cwd "/usr/local/deckbrew"
  user 'postgres'
end

# Upstart
template "deckbrew-api.conf" do
  path "/etc/init/deckbrew-api.conf"
  source "deckbrew-api.conf.erb"
end

template "deckbrew-cache.conf" do
  path "/etc/init/deckbrew-cache.conf"
  source "deckbrew-cache.conf.erb"
end

service "deckbrew-api" do
  provider Chef::Provider::Service::Upstart
  action [:enable, :start]
end

service "deckbrew-cache" do
  provider Chef::Provider::Service::Upstart
  action [:enable, :start]
end
