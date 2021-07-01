require 'securerandom'
require 'faker'

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

                it 'supports concurrent streams' do
                    ids = []
                    
                    3.times do |i|
                        resp = @session.cmd('STREAM', 'W', "write-#{i}.txt")
                        expect(resp).to be_a(Wire::Integer)
                        ids.append(resp.value)
                    end

                    ids.each do |id|
                        @session.put_stream(id)
                        @session.put_blob("hello\n")
                    end

                    ids.each_with_index do |id, i|
                        @session.put_stream(id)
                        @session.put_blob("#{i}\n")
                    end

                    ids.each do |id|
                        @session.put_stream(id)
                        @session.put_null
                    end

                    50.times do
                        resp = @session.cmd('LIST', 'write-2.txt')

                        if resp.is_a? Wire::Error
                            sleep 0.020
                        else
                            break
                        end
                    end

                    3.times do |i|
                        contents = @session.read_file("write-#{i}.txt")
                        expect(contents).to eq("hello\n#{i}\n")
                    end
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

                it 'supports concurrent streams' do
                    Faker::Config.random = Random.new(42)

                    data1 = Faker::Lorem.paragraph * 1000
                    @session.write_file('read-1.txt', data1)

                    data2 = Faker::Lorem.paragraph * 1000
                    @session.write_file('read-2.txt', data2)

                    @session.buffer do |b|
                        b.put_array('STREAM', 'R', "read-1.txt")
                        b.put_array('STREAM', 'R', "read-2.txt")
                    end

                    id1 = nil
                    contents1 = ''

                    id2 = nil
                    contents2 = ''

                    done = 0

                    while done < 2
                        resp = @session.get_next()

                        if resp.is_a? Wire::Integer
                            if id1 == nil
                                id1 = resp.value
                            else
                                id2 = resp.value
                            end

                            next
                        end

                        if !(resp.is_a? Wire::Frame)
                            raise 'response was expected to be a stream frame'
                        end

                        if resp.id != id1 && resp.id != id2
                            raise "unexpected frame id #{resp.id}"
                        end

                        if resp.payload.is_a? Wire::Null
                            # make sure we got at least one blob from each file
                            expect(contents1.length).to be > 0
                            expect(contents2.length).to be > 0

                            done = done + 1
                            next
                        end

                        if !(resp.payload.is_a? Wire::Blob)
                            raise 'expected stream frame to be null or blob'
                        end

                        if resp.id == id1
                            contents1 << resp.payload.value
                        else
                            contents2 << resp.payload.value
                        end
                    end

                    expect(contents1 == data1).to be(true)
                    expect(contents2 == data2).to be(true)
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

    context 'unauthenticated' do
        before(:all) do
            @filename = "test-#{SecureRandom.hex}.txt"
            admin.write_file(@filename, "hello\nstream\n")
        end

        context 'read' do
            it 'returns DENIED' do
                resp = unauth.cmd('STREAM', 'R', @filename)
                expect(resp).to be_error('DENIED')
            end
        end

        context 'write' do
            it 'returns DENIED' do
                resp = unauth.cmd('STREAM', 'W', @filename)
                expect(resp).to be_error('DENIED')
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