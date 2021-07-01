require 'securerandom'

RSpec.describe 'MKDIR' do
    ['single user', 'admin', 'regular user'].each do |persona|
        context "as #{persona}" do
            before(:all) do
                @session = as(persona)
                @dirname = "world-#{SecureRandom.hex}"
                @resp = @session.cmd('MKDIR', @dirname)
            end

            it 'returns OK' do
                expect(@resp).to be_ok
            end

            it 'creates a folder' do
                resp = @session.cmd('LIST', '/')
                expect(resp).to be_a(Wire::Table)

                row = resp.rows.find { |r| r[1].value == @dirname }
                expect(row).to be_a(Array)
            end
        end
    end

    context 'unauthenticated' do
        before(:all) do
            @dirname = "/world/#{SecureRandom.hex}"
            @resp = unauth.cmd('MKDIR', @dirname)
        end

        it 'returns DENIED' do
            expect(@resp).to be_error('DENIED')
        end

        it 'does not create folder' do
            resp = admin.cmd('LIST', '/')
            expect(resp).to be_a(Wire::Table)

            row = resp.rows.find { |r| r[1].value == @dirname }
            expect(row).to be(nil)
        end
    end

    context 'unauthorized' do
        before(:all) do
            @username = Username.get_next
            admin.cmd!('ADDUSER', @username, 'password')

            @session = Session.new
            @session.cmd!('AUTH', 'PWD', @username, 'password')
        end

        context 'implicit deny' do
            before(:all) do
                admin.cmd!('MKDIR', "/home/#{@username}")
                @resp = @session.cmd('MKDIR', "/home/#{@username}/project")
            end

            it 'returns DENIED' do
                expect(@resp).to be_error('DENIED')
            end

            it 'does not create folder' do
                resp = admin.cmd('LIST', "/home/#{@username}")
                expect(resp).to be_a(Wire::Table)

                row = resp.rows.find { |r| r[1].value == 'project' }
                expect(row).to be(nil)
            end
        end

        context 'explicit deny' do
            before(:all) do
                admin.cmd!('MKDIR', "/home2/#{@username}")
                admin.cmd!('PUTACP', "policy-#{SecureRandom.hex}", 'DENY', 'W', [@username], ["/home2"])
                admin.cmd!('PUTACP', "policy-#{SecureRandom.hex}", 'ALLOW', 'W', [@username], ["/home2/#{@username}"])

                @resp = @session.cmd('MKDIR', "/home2/#{@username}/project")
            end

            it 'returns DENIED' do
                expect(@resp).to be_error('DENIED')
            end

            it 'does not create folder' do
                resp = admin.cmd('LIST', "/home2/#{@username}")
                expect(resp).to be_a(Wire::Table)

                row = resp.rows.find { |r| r[1].value == 'project' }
                expect(row).to be(nil)
            end
        end
    end
end