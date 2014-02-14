apt_repository 'nginx' do
  uri          'http://ppa.launchpad.net/nginx/stable/ubuntu'
  distribution node['lsb']['codename']
  components   ['main']
  keyserver    'keyserver.ubuntu.com'
  key          'C300EE8C'
end

package 'nginx'

link "/etc/nginx/sites-enabled/default" do
  action :delete
end

template "image-proxy" do
  path "/etc/nginx/sites-available/image-proxy"
  source "image-proxy.erb"
  mode 0755
  notifies :restart, "service[nginx]"
end

link "/etc/nginx/sites-enabled/image-proxy" do
  to "/etc/nginx/sites-available/image-proxy"
  notifies :restart, "service[nginx]"
end

service 'nginx' do
  action [:enable, :start]
  supports :restart => true
end


