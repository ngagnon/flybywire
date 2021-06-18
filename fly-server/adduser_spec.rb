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

