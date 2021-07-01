RSpec.describe 'RMUSER' do
    context 'admin' do
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

    ['unauthenticated', 'regular user'].each do |persona|
        context "as #{persona}" do
            before(:all) do
                @username = Username.get_next
                admin.cmd!('ADDUSER', @username, 'supersecret')

                @resp = as(persona).cmd('RMUSER', @username)
            end

            it 'returns DENIED' do
                expect(@resp).to be_error('DENIED')
            end

            it 'does not remove user' do
                resp = admin.cmd!('SHOWUSER', @username)
                expect(resp).to be_a(Wire::Map)
                expect(resp['admin']).to be_a(Wire::Boolean)
                expect(resp['admin'].value).to be(false)
            end
        end
    end

    context 'single-user' do
        it 'returns ILLEGAL' do
            username = Username.get_next
            admin.cmd!('ADDUSER', username, 'supersecret')

            resp = single_user.cmd('RMUSER', username)
            expect(resp).to be_error('ILLEGAL')
        end
    end
end

