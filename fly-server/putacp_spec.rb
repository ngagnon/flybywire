require 'securerandom'

RSpec.describe 'PUTACP' do
    context 'as admin' do
        context 'add new policy' do
            before(:all) do
                @username = Username.get_next
                admin.cmd!('ADDUSER', @username, 'password')
                admin.cmd!('MKDIR', "/home/#{@username}/project")
                admin.write_file("/home/#{@username}/project/readme.txt", "hello\nacp\n")

                @rulename = "policy-#{SecureRandom.hex}"
                @resp = admin.cmd('PUTACP', @rulename, 'ALLOW', 'R', [@username], ["/home/#{@username}"])
            end

            it 'returns OK' do
                expect(@resp).to be_ok
            end

            it 'adds policy' do
                resp = admin.cmd('LISTACP')
                policy = resp.rows.find {|e| e[0].value == @rulename}
                expect(policy).to be_a(Array)
                expect(policy[1].value).to eq('ALLOW')
                expect(policy[2].value).to eq('R')
                expect(policy[3].elems.length).to eq(1)
                expect(policy[3].elems[0].value).to eq(@username)
                expect(policy[4].elems.length).to eq(1)
                expect(policy[4].elems[0].value).to eq("/home/#{@username}")
            end

            it 'applies policy' do
                s = Session.new
                s.cmd!('AUTH', 'PWD', @username, 'password')
                contents = s.read_file("/home/#{@username}/project/readme.txt")
                s.close
                expect(contents).to eq("hello\nacp\n")
            end
        end

        context 'update existing policy' do
            before(:all) do
                @username = Username.get_next
                admin.cmd!('ADDUSER', @username, 'password')
                admin.cmd!('MKDIR', "/home/#{@username}/project")

                admin.cmd!('PUTACP', "policy-#{SecureRandom.hex}", 'ALLOW', 'W', [@username], ["/home/#{@username}"])

                @rulename = "policy-#{SecureRandom.hex}"
                admin.cmd!('PUTACP', @rulename, 'DENY', 'W', [@username], ["/usr/home/#{@username}/project"])
                @resp = admin.cmd('PUTACP', @rulename, 'ALLOW', 'R', [@username], ["/home/#{@username}"])
            end

            it 'returns OK' do
                expect(@resp).to be_ok
            end

            it 'updates policy' do
                resp = admin.cmd('LISTACP')

                num_policies = resp.rows.count {|e| e[0].value == @rulename}
                expect(num_policies).to eq(1)

                policy = resp.rows.find {|e| e[0].value == @rulename}
                expect(policy).to be_a(Array)
                expect(policy[1].value).to eq('ALLOW')
                expect(policy[2].value).to eq('R')
                expect(policy[3].elems.length).to eq(1)
                expect(policy[3].elems[0].value).to eq(@username)
                expect(policy[4].elems.length).to eq(1)
                expect(policy[4].elems[0].value).to eq("/home/#{@username}")
            end

            it 'applies policy' do
                s = Session.new
                s.cmd!('AUTH', 'PWD', @username, 'password')
                s.write_file("/home/#{@username}/project/readme.txt", "hello\nacp\n")
                contents = s.read_file("/home/#{@username}/project/readme.txt")
                s.close
                expect(contents).to eq("hello\nacp\n")
            end
        end
    end

    context 'single user' do
        it 'returns ILLEGAL' do
            username = Username.get_next
            resp = single_user.cmd('PUTACP', "policy-#{SecureRandom.hex}", 'ALLOW', 'R',  [username],["/home/#{username}"])
            expect(resp).to be_error('ILLEGAL')
        end
    end

    ['regular user', 'unauthenticated'].each do |persona|
        context "as #{persona}" do
            it 'returns DENIED' do
                username = Username.get_next
                admin.cmd!('ADDUSER', username, 'password')

                session = as(persona)
                resp = session.cmd('PUTACP', "policy-#{SecureRandom.hex}", 'ALLOW', 'R', [username], ["/home/#{username}"])
                expect(resp).to be_error('DENIED')
            end
        end
    end
end