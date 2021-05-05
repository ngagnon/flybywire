RSpec.describe 'File commands' do
    describe 'MKDIR' do
        before(:all) do
            $admin.put_array('MKDIR', 'world')
            @line = $admin.get_string
        end

        it 'returns OK' do
            expect(@line).to eq('OK')
        end

        it 'creates a folder' do
            newdir = File.join($dir, 'world')
            expect(Dir.exist? newdir).to be true
        end
    end
end