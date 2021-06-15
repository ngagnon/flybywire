# @TODO: password cannot be empty + minimum password length?
RSpec.describe 'SETPWD' do
    context 'admin' do
        before(:all) do
            @username = Username.get_next
            admin.cmd!('ADDUSER', @username, 'supersecret')
            @resp = admin.cmd('SETPWD', @username, 'newpassword')
        end

        it 'returns OK' do
            expect(@resp).to be_ok
        end

        it 'changes password' do
            s = Session.new
            s.cmd!('AUTH', 'PWD', @username, 'newpassword')
            s.close
        end
    end

    context 'regular user' do
        context 'self' do
            before(:all) do
                @resp = regular_user.cmd('SETPWD', 'joe', 'newpassword')
            end

            after(:all) do
                # put back the original password
                regular_user.cmd!('SETPWD', 'joe', 'regularguy')
            end

            it 'returns OK' do
                expect(@resp).to be_ok
            end

            it 'updates password' do
                regular_user.cmd!('AUTH', 'PWD', 'joe', 'newpassword')
            end
        end

        context 'someone else' do
            before(:all) do
                @username = Username.get_next
                admin.cmd!('ADDUSER', @username, 'topsecret')

                @resp = regular_user.cmd('SETPWD', @username, 'newpassword')
            end

            it 'returns an error' do
                expect(@resp).to be_a(Wire::Error)
                expect(@resp.code).to eq('DENIED')
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
        before(:all) do
            @username = Username.get_next
            @resp = single_user.cmd('SETPWD', @username, 'newpassword')
        end

        it 'returns an error' do
            expect(@resp).to be_a(Wire::Error)
            expect(@resp.code).to eq('ILLEGAL')
        end
    end
end

