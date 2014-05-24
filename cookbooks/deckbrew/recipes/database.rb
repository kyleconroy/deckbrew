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
package 'mercurial'
package 'git'
package 'varnish'
package 'postgresql-9.3'
package 'postgresql-contrib-9.3'
package 'unzip'

directory "/usr/local/gopath"

template "go-profile" do
  path "/etc/profile.d/go.sh"
  source "goprofile.erb"
  mode 0755
end

tar_extract 'https://go.googlecode.com/files/go1.2.linux-amd64.tar.gz' do
  target_dir '/usr/local'
  creates '/usr/local/go/bin'
end

service "varnish" do
  action :enable
  supports :status => true, :start => true, :stop => true, :restart => true
end

template "varnish" do
  path "/etc/varnish/default.vcl"
  source "default.vcl.erb"
  notifies :restart, "service[varnish]"
end

template "deckcache-defaults" do
  path "/etc/default/varnish"
  source "deckbrew-cache.conf.erb"
  mode 0755
  notifies :restart, "service[varnish]"
end

GO = {
  "PATH" => "#{ENV['PATH']}:/usr/local/go/bin",
  "GOPATH" => "/usr/local/gopath",
  "DATABASE_USER" => node['deckbrew']['database']['user'],
  "DATABASE_HOST" => node['deckbrew']['database']['host'],
  "DATABASE_PASSWORD" => node['deckbrew']['database']['password'],
  "DECKBREW_HOSTNAME" => node['deckbrew']['hostname'],
}

directory 'gopath'

execute 'make clean deps brewapi' do
  cwd '/usr/local/deckbrew'
  environment (GO)
end

# Create the database
execute 'make syncdb' do
  cwd '/usr/local/deckbrew'
  user 'postgres'
  environment (GO)
end

execute 'make prices.json' do
  cwd '/usr/local/deckbrew'
  environment (GO)
end


# Upstart
template "deckapi" do
  path "/etc/init/deckapi.conf"
  source "deckbrew-api.conf.erb"
  notifies :restart, "service[deckapi]"
end

service "deckapi" do
  provider Chef::Provider::Service::Upstart
  action [:enable, :start]
  supports :status => true, :start => true, :stop => true, :restart => true
end
