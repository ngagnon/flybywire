require 'securerandom'

RSpec.describe 'CHROOT' do
    context 'as admin' do
        before(:all) do
            @username = Username.get_next
            admin.cmd!('ADDUSER', @username, 'secret')
            admin.cmd!('PUTACP', "policy-#{SecureRandom.hex}", 'ALLOW', 'R', [@username], ['/'])
            admin.cmd!('PUTACP', "policy-#{SecureRandom.hex}", 'ALLOW', 'W', [@username], ['/'])

            @dir = "/chroot-#{SecureRandom.hex}"
            admin.cmd!('MKDIR', @dir)

            @filename = "file-#{SecureRandom.hex}.txt"
            admin.write_file("#{@dir}/#{@filename}", "hello\nchroot\n")
            
            @resp = admin.cmd('CHROOT', @username, @dir)
        end

        it 'returns OK' do
            expect(@resp).to be_ok
        end

        it 'sets chroot property' do
            resp = admin.cmd('SHOWUSER', @username)
            expect(resp['chroot'].value).to eq(@dir)
        end

        it 'changes user virtual root' do
            s = Session.new
            s.cmd!('AUTH', 'PWD', @username, 'secret')

            resp = s.cmd!('LIST', '/')
            expect(resp.row_count).to eq(1)
            expect(resp[0][1].value).to eq(@filename)

            contents = s.read_file(@filename)
            expect(contents).to eq("hello\nchroot\n")

            resp = s.cmd('LIST', "#{@dir}/#{@filename}")
            expect(resp).to be_error('NOTFOUND')

            resp = s.cmd('STREAM', 'R', "#{@dir}/#{@filename}")
            expect(resp).to be_error('NOTFOUND')

            s.close
        end
    end

    context 'single user' do
        it 'returns ILLEGAL' do
            session = as('single user')

            dir = "chroot-#{SecureRandom.hex}"
            session.cmd!('MKDIR', dir)

            username = Username.get_next
            resp = session.cmd('CHROOT', username, dir)
            expect(resp).to be_error('ILLEGAL')
        end
    end

    ['regular user', 'unauthenticated'].each do |persona|
        context "as #{persona}" do
            it 'returns DENIED' do
                username = Username.get_next
                admin.cmd!('ADDUSER', username, 'secret')

                dir = "chroot-#{SecureRandom.hex}"
                admin.cmd!('MKDIR', dir)

                session = as(persona)
                resp = session.cmd('CHROOT', username, dir)
                expect(resp).to be_error('DENIED')
            end
        end
    end
end