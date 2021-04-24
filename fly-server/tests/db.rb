require 'bcrypt'
require 'csv'

class FlyDB
    def self.setup(dir)
        write_version(dir)
        write_user_db(dir)
        write_acp_db(dir)
    end

    def self.write_version(dir)
        File.write(File.join(dir, '.fly/version'), '1')
    end

    def self.write_user_db(dir)
        path = File.join(dir, '.fly/users.csv')

        CSV.open(path, "w") do |csv|
            csv << ["username", "password", "chroot", "admin"]
        
            pwd = BCrypt::Password.create("secret")
            csv << ["admin", pwd, "", "1"]
        end
    end

    def self.write_acp_db(dir)
        path = File.join(dir, '.fly/acp.csv')

        CSV.open(path, "w") do |csv|
            csv << ["rule", "users", "paths", "permissions"]
            csv << ["default", "*", "/", "WW"]
        end
    end
end