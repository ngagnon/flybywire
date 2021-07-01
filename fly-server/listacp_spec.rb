RSpec.describe 'LISTACP' do
    context 'as admin' do
        before(:all) do
            @username = Username.get_next
            admin.cmd!('ADDUSER', @username, 'password')

            @rule_name = "policy-#{SecureRandom.hex}"
            admin.cmd!('PUTACP', @rule_name, 'ALLOW', 'R', [@username], ["/home/#{@username}"])

            @resp = admin.cmd('LISTACP')
        end

        it 'returns list of policies' do
            expect(@resp).to be_a(Wire::Table)
            expect(@resp.col_count).to eq(5)

            @resp.each do |e|
                expect(e[0]).to be_a(Wire::String)
                expect(e[1]).to be_a(Wire::String)
                expect(e[2]).to be_a(Wire::String)
                expect(e[3]).to be_a(Wire::Array)
                expect(e[4]).to be_a(Wire::Array)
            end

            policy = @resp.rows.find {|e| e[0].value == @rule_name}
            expect(policy).to be_a(Array)

            expect(policy[0].value).to eq(@rule_name)
            expect(policy[1].value).to eq('ALLOW')
            expect(policy[2].value).to eq('R')
            expect(policy[3].elems.length).to eq(1)
            expect(policy[3].elems[0].value).to eq(@username)
            expect(policy[4].elems.length).to eq(1)
            expect(policy[4].elems[0].value).to eq("/home/#{@username}")
        end
    end

    context 'single user' do
        it 'returns ILLEGAL' do
            resp = single_user.cmd('LISTACP')
            expect(resp).to be_error('ILLEGAL')
        end
    end

    ['regular user', 'unauthenticated'].each do |persona|
        context "as #{persona}" do
            it 'returns DENIED' do
                session = as(persona)
                resp = session.cmd('LISTACP')
                expect(resp).to be_error('DENIED')
            end
        end
    end
end