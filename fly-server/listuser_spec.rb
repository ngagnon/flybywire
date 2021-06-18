# @TODO: all user commands should return errors in single-user mode
# @TODO: the commands shouldn't work if you're not admin (or unauth)
# @TODO: make sure changes are persisted after server restart? (separate test)
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