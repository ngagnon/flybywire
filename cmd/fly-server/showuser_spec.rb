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

