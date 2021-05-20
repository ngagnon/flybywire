RSpec.describe 'STREAM' do
    context 'authorized' do
        describe 'write' do
            before(:all) do
                $admin.put_array('STREAM', 'W', 'test.txt')
                (@type, @id) = $admin.get_next
            end

            it 'returns stream id' do
                expect(@type).to be(:int)
            end

            it 'ignores frames with invalid stream ID' do
                $admin.put_stream(2)
                $admin.pub_blob("hello1\n")

                $admin.put_stream(1000000)
                $admin.pub_blob("hello2\n")

                $admin.put_array('PING')
                line = $admin.get_string
                expect(line).to eq('PONG')
            end

            # @TODO: concurrent streams

            it 'writes to file' do
                $admin.put_stream(@id)
                $admin.pub_blob("hello1\n")

                $admin.put_stream(@id)
                $admin.pub_blob("hello2\n")

                $admin.put_stream(@id)
                $admin.pub_blob("hello3\n")

                $admin.put_stream(@id)
                $admin.put_null

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
    end
end