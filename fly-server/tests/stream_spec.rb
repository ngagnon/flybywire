RSpec.describe 'STREAM' do
    context 'authorized' do
        describe 'for writing' do
            before(:all) do
                $admin.put_array('STREAM', 'W', 'test.txt')
                (@type, @id) = $admin.get_next
            end

            it 'returns stream id' do
                expect(@type).to be(:int)
            end

            # @TODO: concurrent streams
            # @TODO: handles streams that are way too big (e.g. 1000000)
            # @TODO: handles streams that haven't been open yet (e.g. 2)

            it 'writes to file' do
                $admin.put_stream(@id)
                $admin.pub_blob("hello1\n")

                $admin.put_stream(@id)
                $admin.pub_blob("hello2\n")

                $admin.put_stream(@id)
                $admin.pub_blob("hello3\n")

                $admin.put_stream(@id)
                $admin.put_null
                sleep 0.100

                filepath = File.join($dir, 'test.txt')
                content = File.read(filepath)
                expect(content).to eq "hello1\nhello2\nhello3\n"
            end
        end
    end
end