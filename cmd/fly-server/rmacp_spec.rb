require 'securerandom'

RSpec.describe 'RMACP' do
    context 'as admin' do
        before(:all) do
            @username = Username.get_next
            admin.cmd!('ADDUSER', @username, 'password')
            admin.cmd!('MKDIR', "/home/#{@username}")
            admin.write_file("/home/#{@username}/readme.txt", "hello\nacp\n")

            @rulename = "policy-#{SecureRandom.hex}"
            admin.cmd!('PUTACP', @rulename, 'ALLOW', 'R', [@username], ["/home/#{@username}"])
            @resp = admin.cmd('RMACP', @rulename)
        end

        it 'returns OK' do
            expect(@resp).to be_ok
        end

        it 'removes policy' do
            resp = admin.cmd('LISTACP')
            policy = resp.rows.find {|e| e[0].value == @rulename}
            expect(policy).to be(nil)
        end

        it 'stops enforcing policy' do
            s = Session.new
            s.cmd!('AUTH', 'PWD', @username, 'password')
            resp = s.cmd('STREAM', 'R', "/home/#{@username}/readme.txt")
            s.close
            expect(resp).to be_error('DENIED')
        end
    end

    context 'single user' do
        it 'returns ILLEGAL' do
            username = Username.get_next
            resp = single_user.cmd('RMACP', "policy-#{SecureRandom.hex}")
            expect(resp).to be_error('ILLEGAL')
        end
    end

    ['regular user', 'unauthenticated'].each do |persona|
        context "as #{persona}" do
            it 'returns DENIED' do
                username = Username.get_next
                admin.cmd!('ADDUSER', username, 'password')

                rulename = "policy-#{SecureRandom.hex}"
                admin.cmd!('PUTACP', rulename, 'ALLOW', 'R', [username], ["/home/#{username}"])

                session = as(persona)
                resp = session.cmd('RMACP', rulename)
                expect(resp).to be_error('DENIED')
            end
        end
    end
end