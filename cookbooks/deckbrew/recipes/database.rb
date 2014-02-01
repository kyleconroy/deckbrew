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
package 'unzip'

tar_extract 'https://go.googlecode.com/files/go1.2.linux-amd64.tar.gz' do
  target_dir '/usr/local'
  creates '/usr/local/go/bin'
end


template "varnish" do
  path "/etc/varnish/default.vcl"
  source "default.vcl.erb"
end

GO = {
  "PATH" => "#{ENV['PATH']}:/usr/local/go/bin",
  "GOPATH" => "/usr/local/gopath",
}

directory 'gopath'

# Build the binary
execute 'make deps' do
  cwd '/usr/local/deckbrew'
  environment (GO)
end

execute 'make' do
  cwd '/usr/local/deckbrew'
  environment (GO)
end

# Create the database
execute 'make syncdb' do
  cwd '/usr/local/deckbrew'
  user 'postgres'
  environment (GO)
end

# Upstart
template "deckapi" do
  path "/etc/init/deckapi.conf"
  source "deckbrew-api.conf.erb"
end

template "deckcache-defaults" do
  path "/etc/default/varnish"
  source "deckbrew-cache.conf.erb"
end

service "deckapi" do
  provider Chef::Provider::Service::Upstart
  action [:enable, :start]
end

service "varnish" do
  action [:enable, :start]
end
