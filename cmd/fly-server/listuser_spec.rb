RSpec.describe 'LISTUSER' do
    context 'admin' do
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

    ['unauthenticated', 'regular user'].each do |persona|
        context "as #{persona}" do
            it 'returns DENIED' do
                resp = as(persona).cmd('LISTUSER')
                expect(resp).to be_error('DENIED')
            end
        end
    end

    context 'single-user' do
        it 'returns ILLEGAL' do
            resp = single_user.cmd('LISTUSER')
            expect(resp).to be_error('ILLEGAL')
        end
    end
end