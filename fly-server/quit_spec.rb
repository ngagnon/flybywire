RSpec.describe 'QUIT' do
    context 'authenticated' do
        before(:each) do
            @session = Session.new
            @session.cmd!('AUTH', 'PWD', 'example', 'supersecret')
        end

        after(:each) do
            @session.close
        end

        it 'returns OK' do
            resp = @session.cmd('QUIT')
            expect(resp).to be_a(Wire::String)
            expect(resp.value).to eq('OK')
        end

        it 'cancels all pipelined commands' do
            @session.buffer do |b|
                b.put_array("MKDIR", "quit-hello")
                b.put_array("QUIT")
                b.put_array("MKDIR", "quit-world")
            end

            @session.get_string
            @session.get_string

            newdir = File.join($dir, 'quit-hello')
            expect(Dir.exist? newdir).to be true

            newdir = File.join($dir, 'quit-world')
            expect(Dir.exist? newdir).to be false
        end
    end

    context 'unauthenticated' do
        before(:each) do
            @session = Session.new
        end

        after(:each) do
            @session.close
        end

        it 'returns OK' do
            resp = @session.cmd('QUIT')
            expect(resp).to be_a(Wire::String)
            expect(resp.value).to eq('OK')
        end
    end
end
