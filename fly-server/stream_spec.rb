require 'securerandom'

RSpec.describe 'STREAM' do
    ['admin', 'regular user'].each do |persona|
        context "as #{persona}" do
            describe 'write' do
                before(:all) do
                    @session = as(persona)
                    @resp = @session.cmd('STREAM', 'W', 'test.txt')
                    @id = @resp.value
                end

                it 'returns stream id' do
                    expect(@resp).to be_a(Wire::Integer)
                end

                it 'ignores frames with invalid stream ID' do
                    @session.put_stream(2)
                    @session.put_blob("hello1\n")
                    resp = @session.get_next
                    expect(resp).to be_a(Wire::Frame)
                    expect(resp.id).to eq(2)
                    expect(resp.payload).to be_a(Wire::Error)
                    expect(resp.payload.msg).to include('closed')

                    @session.put_stream(1000000)
                    @session.put_blob("hello2\n")
                    resp = @session.get_next
                    expect(resp).to be_a(Wire::Frame)
                    expect(resp.id).to eq(1000000)
                    expect(resp.payload).to be_a(Wire::Error)
                    expect(resp.payload.msg).to include('closed')
                end

                # @TODO: concurrent streams

                it 'writes to file' do
                    @session.put_stream(@id)
                    @session.put_blob("hello1\n")

                    @session.put_stream(@id)
                    @session.put_blob("hello2\n")

                    @session.put_stream(@id)
                    @session.put_blob("hello3\n")

                    @session.put_stream(@id)
                    @session.put_null

                    filepath = File.join($dir, 'test.txt')
                    i = 10

                    until File.exists?(filepath) || i == 0
                        sleep 0.100
                        i = i - 1
                    end

                    content = File.read(filepath)
                    expect(content).to eq "hello1\nhello2\nhello3\n"
                end
            end

            describe 'read' do
                before(:all) do
                    admin.write_file('test-read.txt', "hello1\nhello2\nhello3\nfoobar\n")

                    @session = as(persona)
                    @resp = @session.cmd!('STREAM', 'R', 'test-read.txt')
                    @id = @resp.value
                end

                it 'returns stream id' do
                    expect(@resp).to be_a(Wire::Integer)
                end

                # @TODO: concurrent streams

                it 'sends chunks' do
                    resp = @session.get_next
                    expect(resp).to be_a(Wire::Frame)
                    expect(resp.id).to eq(@id)
                    
                    payload = resp.payload
                    expect(payload).to be_a(Wire::Blob)
                    expect(payload.value).to eq("hello1\nhello2\nhello3\nfoobar\n")

                    resp = @session.get_next
                    expect(resp).to be_a(Wire::Frame)
                    expect(resp.id).to eq(@id)
                    expect(resp.payload).to be_a(Wire::Null)
                end

                it 'returns NOTFOUND when file does not exist' do
                    resp = @session.cmd('STREAM', 'R', 'test-not-exist.txt')
                    expect(resp).to be_a(Wire::Error)
                    expect(resp.code).to eq('NOTFOUND')
                end

                it 'returns NOTFOUND for internal files' do
                    resp = @session.cmd('STREAM', 'R', '.fly/users.csv')
                    expect(resp).to be_a(Wire::Error)
                    expect(resp.code).to eq('NOTFOUND')
                end
            end
        end
    end

    context 'unauthorized' do
        context 'implicit deny' do
            before(:all) do
                @username = Username.get_next
                admin.cmd!('ADDUSER', @username, 'password')
                admin.cmd!('MKDIR', "/home/#{@username}")
                admin.write_file("/home/#{@username}/test.txt", "hello\nimplicit\ndeny\n")

                # On different path
                admin.cmd!('PUTACP', "policy-#{SecureRandom.hex}", 'ALLOW', 'R', [@username], ["/usr/home/#{@username}"])
                admin.cmd!('PUTACP', "policy-#{SecureRandom.hex}", 'ALLOW', 'W', [@username], ["/usr/home/#{@username}"])

                # On different user
                admin.cmd!('PUTACP', "policy-#{SecureRandom.hex}", 'ALLOW', 'R', ['u' + @username], ["/home/#{@username}"])
                admin.cmd!('PUTACP', "policy-#{SecureRandom.hex}", 'ALLOW', 'W', ['u' + @username], ["/home/#{@username}"])

                @session = Session.new
                @session.cmd!('AUTH', 'PWD', @username, 'password')
            end

            after(:all) do
                @session.close
            end

            context 'read' do
                it 'returns DENIED' do
                    resp = @session.cmd('STREAM', 'R', "/home/#{@username}/test.txt")
                    expect(resp).to be_error('DENIED')
                end
            end

            context 'write' do
                it 'returns DENIED' do
                    resp = @session.cmd('STREAM', 'W', "/home/#{@username}/test.txt")
                    expect(resp).to be_error('DENIED')
                end
            end
        end

        context 'explicit deny' do
            before(:all) do
                @username = Username.get_next
                admin.cmd!('ADDUSER', @username, 'password')
                admin.cmd!('MKDIR', "/home/#{@username}/some/project")
                admin.write_file("/home/#{@username}/some/project/test.txt", "hello\nexplicit\ndeny\n")

                # Reads
                admin.cmd!('PUTACP', "policy-#{SecureRandom.hex}", 'ALLOW', 'R', [@username], ["/home/#{@username}"])
                admin.cmd!('PUTACP', "policy-#{SecureRandom.hex}", 'DENY', 'R', [@username], ["/home/#{@username}/some"])
                admin.cmd!('PUTACP', "policy-#{SecureRandom.hex}", 'ALLOW', 'R', [@username], ["/home/#{@username}/some/project"])

                # Writes
                admin.cmd!('PUTACP', "policy-#{SecureRandom.hex}", 'ALLOW', 'W', [@username], ["/home/#{@username}"])
                admin.cmd!('PUTACP', "policy-#{SecureRandom.hex}", 'DENY', 'W', [@username], ["/home/#{@username}/some"])
                admin.cmd!('PUTACP', "policy-#{SecureRandom.hex}", 'ALLOW', 'W', [@username], ["/home/#{@username}/some/project"])

                @session = Session.new
                @session.cmd!('AUTH', 'PWD', @username, 'password')
            end

            after(:all) do
                @session.close
            end

            context 'read' do
                it 'returns DENIED' do
                    resp = @session.cmd('STREAM', 'R', "/home/#{@username}/some/project/test.txt")
                    expect(resp).to be_error('DENIED')
                end
            end

            context 'write' do
                it 'returns DENIED' do
                    resp = @session.cmd('STREAM', 'W', "/home/#{@username}/some/project/test.txt")
                    expect(resp).to be_error('DENIED')
                end
            end
        end
    end
end