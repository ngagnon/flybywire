RSpec.describe 'SETADM' do
    context 'admin' do
        before(:all) do
            @username = Username.get_next
            admin.cmd!('ADDUSER', @username, 'supersecret')
            @resp = admin.cmd('SETADM', @username, true)
        end

        it 'returns OK' do
            expect(@resp).to be_ok
        end

        it 'makes user admin' do
            resp = admin.cmd!('SHOWUSER', @username)
            expect(resp).to be_a(Wire::Map)
            expect(resp['admin']).to be_a(Wire::Boolean)
            expect(resp['admin'].value).to be(true)
        end
    end

    context 'regular user' do
        before(:all) do
            @resp = regular_user.cmd('SETADM', 'joe', true)
        end

        it 'returns an error' do
            expect(@resp).to be_error('DENIED')
        end

        it 'does not change admin' do
            resp = admin.cmd!('SHOWUSER', 'joe')
            expect(resp).to be_a(Wire::Map)
            expect(resp['admin']).to be_a(Wire::Boolean)
            expect(resp['admin'].value).to be(false)
        end
    end

    context 'unauthenticated' do
        before(:all) do
            @resp = unauth.cmd('SETADM', 'joe', true)
        end

        it 'returns an error' do
            expect(@resp).to be_error('DENIED')
        end

        it 'does not change admin' do
            resp = admin.cmd!('SHOWUSER', 'joe')
            expect(resp).to be_a(Wire::Map)
            expect(resp['admin']).to be_a(Wire::Boolean)
            expect(resp['admin'].value).to be(false)
        end
    end

    context 'single-user' do
        it 'returns an error' do
            resp = single_user.cmd('SETADM', 'example', false)
            expect(resp).to be_error('ILLEGAL')
        end
    end
end

