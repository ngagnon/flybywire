RSpec.describe 'TOKEN' do
    context 'authenticated' do
        ['admin', 'regular user'].each do |persona|
            context "as #{persona}" do
                before(:all) do
                    @session = as(persona)
                    @resp = @session.cmd('TOKEN')
                end

                it 'returns string' do
                    expect(@resp).to be_a(Wire::String)
                    expect(@resp.value.length).to be > 0
                end

                it 'allows you to login as same user' do
                    resp = @session.cmd('WHOAMI')
                    username = resp.value

                    session = Session.new
                    session.cmd!('AUTH', 'TOK', @resp.value)
                    resp = session.cmd('WHOAMI')
                    expect(resp).to be_a(Wire::String)
                    expect(resp.value).to eq(username)
                end
            end
        end
    end

    context 'as unauthenticated' do
        it 'returns DENIED' do
            resp = unauth.cmd('TOKEN')
            expect(resp).to be_error('DENIED')
        end
    end

    context 'single user' do
        it 'returns ILLEGAL' do
            resp = single_user.cmd('TOKEN')
            expect(resp).to be_error('ILLEGAL')
        end
    end
end