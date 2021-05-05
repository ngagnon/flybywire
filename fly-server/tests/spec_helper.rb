require 'resp'
require 'server'

RSpec.configure do |config|
    config.before(:suite) do
        $dir = Dir.mktmpdir 'fly'
        @s = Server.new $dir
        @r = RESP.new

        @r.put_array('ADDUSER', 'example', 'supersecret')
        @r.get_next

        @r.close
        @s.kill

        $s = Server.new $dir

        $admin = RESP.new
        $admin.put_array('AUTH', 'PWD', 'example', 'supersecret')
        line = $admin.get_string

        $unauth = RESP.new
    end

    config.after(:suite) do
        $admin.close
        $unauth.close
        $s.kill
        FileUtils.rm_rf $dir
    end
end