# @TODO: all user commands should return errors in single-user mode
# @TODO: the commands shouldn't work if you're not admin (or unauth)
# @TODO: test SETPWD
# @TODO: SETPWD should work on yourself
# @TODO: make sure changs are persisted after server restart?

RSpec.describe 'SHOWUSER' do
    before(:all) do
        @username = Username.get_next
        admin.cmd!('ADDUSER', @username, 'topsecret')
        @resp = admin.cmd('SHOWUSER', @username)
    end

    it 'returns user' do
        expect(@resp).to be_a(Wire::Map)
        expect(@resp.keys).to match_array(['username', 'chroot', 'admin'])
        expect(@resp['username']).to be_a(Wire::String)
        expect(@resp['chroot']).to be_a(Wire::String)
        expect(@resp['admin']).to be_a(Wire::Boolean)
        expect(@resp['username'].value).to eq(@username)
    end
end

RSpec.describe 'ADDUSER' do
    before(:all) do
        @username = Username.get_next
        @resp = admin.cmd('ADDUSER', @username, 'butler9000')
    end

    it 'returns OK' do
        expect(@resp).to be_a(Wire::String)
        expect(@resp.value).to eq('OK')
    end

    it 'creates user with correct defaults' do
        resp = admin.cmd('SHOWUSER', @username)
        expect(resp).to be_a(Wire::Map)
        expect(resp['username'].value).to eq(@username)
        expect(resp['chroot'].value).to eq('')
        expect(resp['admin'].value).to be(false)
    end
end

RSpec.describe 'RMUSER' do
    before(:all) do
        @usernames = Username.get_next(3)
        @usernames.each do |u|
            admin.cmd!('ADDUSER', u, 'topsecret')
        end

        @resp = admin.cmd('RMUSER', @usernames[1])
    end

    it 'returns OK' do
        expect(@resp).to be_a(Wire::String)
        expect(@resp.value).to eq('OK')
    end

    it 'deletes user' do
        resp = admin.cmd('SHOWUSER', @usernames[1])
        expect(resp).to be_a(Wire::Error)
        expect(resp.code).to eq('NOTFOUND')

        resp = admin.cmd('AUTH', 'PWD', @usernames[1], 'topsecret')
        expect(resp).to be_a(Wire::Error)
        expect(resp.code).to eq('DENIED')
    end
end

RSpec.describe 'SETPWD' do
    context 'admin' do
        # @TODO
    end

    context 'regular user' do
        before(:all) do
            @username = Username.get_next
            admin.cmd!('ADDUSER', @username, 'topsecret')
        end

        context 'self' do
            before(:all) do
            end
            it 'returns OK' do
            end

            it 'updates user' do
                # @TODO: mod user
                # @TODO: test that it returns OK
                # @TODO: show user again
            end
        end

        context 'someone else' do
            before(:all) do
                # @TODO
            end

            it 'returns an error' do
                # @TODO: DENIED
            end
        end
    end

    context 'unauthenticated' do
        before(:all) do
            @username = Username.get_next
            admin.cmd!('ADDUSER', @username, 'topsecret')
            @resp = unauth.cmd('SETPWD', @username, 'newpassword')
        end

        it 'returns an error' do
            expect(@resp).to be_a(Wire::Error)
            expect(@resp.code).to eq('DENIED')
        end

        it 'does not change password' do
            resp = unauth.cmd('AUTH', 'PWD', @username, 'newpassword')
            expect(resp).to be_a(Wire::Error)
            expect(resp.code).to eq('DENIED')
        end
    end

    context 'single-user' do
        # @TODO
    end
end

RSpec.describe 'LISTUSER' do
    before(:all) do
        @usernames = Username.get_next(3)
        @usernames.each do |u|
            admin.cmd!('ADDUSER', u, 'topsecret')
        end
    end

    it 'returns list of usernames' do
        resp = admin.cmd('LISTUSER')
        expect(resp).to be_a(Wire::Array)
        expect(resp.elems.any? { |x| @usernames[0] == x.value }).to be(true)
        expect(resp.elems.any? { |x| @usernames[1] == x.value }).to be(true)
        expect(resp.elems.any? { |x| @usernames[2] == x.value }).to be(true)

        admin.cmd!('RMUSER', @usernames[1])

        resp = admin.cmd('LISTUSER')
        expect(resp.elems.any? { |x| @usernames[1] == x.value }).to be(false)
    end
end