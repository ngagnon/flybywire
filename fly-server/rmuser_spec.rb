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

